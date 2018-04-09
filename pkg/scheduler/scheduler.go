package scheduler

import (
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Scheduler struct {
}

func (s *Scheduler) CacheSchedulerAll(m *manager.Manager) {
	//对所有的人脸集进行调度
	for k, _ := range m.FacesetMap {
		if k != "" {
			s.cacheScheduler(k, m);
		}
	}
}

func (s *Scheduler) cacheScheduler(facesetname string, m *manager.Manager) {
	//分两个任务调度
	var lock sync.RWMutex

	//1.注册缓存
	lock.Lock()
	portal := m.UploadPortal[facesetname]
	tempRegisterCache := portal.TempRegisterCache
	glog.Infof("starting schedule faceset: %s, its cache size is %d", facesetname, len(portal.RegisterCache))
	tempRegisterCache = append(tempRegisterCache, portal.RegisterCache...)
	portal.RegisterCache = nil
	m.UploadPortal[facesetname] = portal
	lock.Unlock()

	if len(tempRegisterCache) > 0 {
		s.addCacheFaceSet(tempRegisterCache, facesetname, m)
		for {
			glog.Infof("for loop: tempRegisterCache size is %d.", len(tempRegisterCache))
			if len(tempRegisterCache) == 0 {
				break
			}

			picSample := &tempRegisterCache[len(tempRegisterCache)-1]
			tempRegisterCache = tempRegisterCache[:len(tempRegisterCache)-1]
			similaryRelations := s.caculateSimilarityWithCache(picSample, tempRegisterCache, picSample.ImageBase64, facesetname+model.CACHE_SUFFIX, m)

			for i := 0; i < len(similaryRelations); i = i + 1 {
				for j := len(tempRegisterCache) - 1; i >= 0; i-- {
					if tempRegisterCache[j].Id == similaryRelations[i].To.Id {
						glog.Infof("delete id %s from cache.", tempRegisterCache[j].Id)
						tempRegisterCache = append(tempRegisterCache[:j], tempRegisterCache[j+1:]...)
					}
				}
			}
		}
		//remove the last element, as know is picSample
		//tempRegisterCache = tempRegisterCache[:len(tempRegisterCache)-1]

	}
	//delate all
	s.deleteCacheFaceSet(tempRegisterCache, facesetname, m)

	//2.识别缓存
	lock.Lock()
	portal = m.UploadPortal[facesetname]
	tempDetectCache := portal.TempDetectCache
	tempDetectCache = append(tempDetectCache, portal.DetectCache...)
	portal.DetectCache = nil
	m.UploadPortal[facesetname] = portal
	lock.Unlock()

	if len(tempDetectCache) > 0 {
		s.addCacheFaceSet(tempDetectCache, facesetname, m)

		srMetrix := model.SRMatrix{}
		lastSaveMap := make(model.LastSaveMap) //全局？

		for _, picSample := range tempDetectCache {
			similaryRelations := s.caculateSimilarityWithCache(&picSample, tempDetectCache, picSample.ImageBase64, facesetname+model.CACHE_SUFFIX, m)
			srMetrix = append(srMetrix, similaryRelations)
		}

		if len(srMetrix) > 0 {
			mostSimiliarityRelations := s.caculateMostSimilarity(&srMetrix)

			tempDetectCache = nil

			for i := 0; i < len(mostSimiliarityRelations); i = i + 1 {
				fromPic := mostSimiliarityRelations[i].From
				toPic := mostSimiliarityRelations[i].To

				if fromPic.UploadTime-lastSaveMap[toPic] > 30*1000 {
					m.SaveToDetectDB(fromPic, facesetname)

					lastSaveMap[toPic] = fromPic.UploadTime
				} else {
					tempDetectCache = append(tempDetectCache, *fromPic)
				}
			}
		}
		s.deleteCacheFaceSet(tempDetectCache, facesetname, m)
	}
}

//计算相似度

//计算当前图片与缓存集的相似度，注意该图片也在缓存集中，计算时需要过滤掉自己与自己的相似度
func (s *Scheduler) caculateSimilarityWithCache(picSample *model.PicSample, cacheList []model.PicSample, imageBase64 string, facesetName string, m *manager.Manager) model.SRList {
	m.CaculateSimilarity(picSample, imageBase64, facesetName)
	result := model.SRList{}

	for k, v := range picSample.Similarity {
		//如果相似度大于99，认为是缓存中存在该图片，需要过滤处理
		if v < 99 {
			to := s.getPicSample(k, cacheList)
			if to != nil {
				temp := model.SimilaryRelation{}
				temp.Similary = v
				temp.From = picSample
				temp.To = to
				result = append(result, temp)
			}
		}
	}
	return result
}

//按id从list中找到该对象
func (m *Scheduler) getPicSample(id string, list []model.PicSample) *model.PicSample {
	if len(list) > 0 {
		for _, v := range list {
			if v.Id == id {
				return &v
			}
		}
	}
	return nil
}

//从相似矩阵获取最相似列表
func (m *Scheduler) caculateMostSimilarity(matrix *model.SRMatrix) model.SRList {
	result := model.SRList{}

	srFromMap := model.SRFromMap{}
	srToMap := model.SRToMap{}
	srMap := model.SRMap{}

	for _, srList := range *matrix {
		for _, sr := range srList {
			if v, ok := srFromMap[sr.From]; ok {
				if sr.Similary > v {
					srFromMap[sr.From] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				}
			} else {
				srFromMap[sr.From] = sr.Similary
				tempToMap := model.SRToMap{}
				tempToMap[sr.To] = sr.Similary
				srMap[sr.From] = tempToMap
			}

			if v, ok := srToMap[sr.To]; ok {
				if sr.Similary > v {
					srToMap[sr.To] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				} else {
					srToMap[sr.To] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				}
			}
		}
	}

	for from, toMap := range srMap {
		for to, similarity := range toMap {
			var sr model.SimilaryRelation
			sr.From = from
			sr.To = to
			sr.Similary = similarity
			result = append(result, sr)
		}
	}
	return result
}

//将缓存的图片集放到云上
func (s *Scheduler) addCacheFaceSet(cacheList []model.PicSample, facesetName string, m *manager.Manager) error {
	m.CreateFacesetIfNotExist(facesetName + model.CACHE_SUFFIX)
	if len(cacheList) > 0 {
		glog.Infof("cacheFaceSet size is %d", len(cacheList))
		for _, picSample := range cacheList {
			// first detect image face
			//picSample, ok := pic.Value.(model.PicSample)
			//if !ok {
			//	glog.Errorf("element in list is not PicSample: ", pic.Value)
			//	return nil
			//}

			imagedecode, err := base64.StdEncoding.DecodeString(picSample.ImageBase64)
			if err != nil {
				glog.Error(err)
				return err
			}
			jdface, err := m.DetectFace(picSample.ImageBase64)
			if err != nil {
				glog.Error(err)
				return err
			}
			glog.Infof("dface: %#v", jdface)

			// save to file
			if jdface != nil {
				imageaddress := m.CustConfig.StaticDir + "/" + facesetName + "/" + picSample.Id
				fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
				if err != nil {
					glog.Error(err)
					return err
				}
				defer fileToSave.Close()
				if _, err := fileToSave.Write(imagedecode); err != nil {
					glog.Errorf("buf copy to file err: %s", err.Error())
					return err
				}

				imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + facesetName + "/" + picSample.Id
				// /v1/faceSet/13345/addFace
				urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetName+model.CACHE_SUFFIX] + "/addFace"
				body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\", \"faceSetId\": \"%s\"}", imageurl, m.FaceidMap[facesetName+model.CACHE_SUFFIX]))
				// resp, err := m.AiCloud.FakeAddFace(urlStr, http.MethodPost, body)
				glog.Infof("addCacheFaceSet: the urlStr is %s, body is %s.", urlStr, body)
				resp, err := m.AiCloud.AddFace(urlStr, http.MethodPut, body)
				if err != nil {
					glog.Errorf("add to faceset err: %s", err.Error())
				}
				data := resp
				// glog.Infof("string:%s, resp :%#v", data, data)
				if len(resp) == 0 {
					glog.Errorf("add face return 0 length")
					return err
				}
				bdata := make(map[string]interface{})
				json.Unmarshal(data, &bdata)

			} else {
				return err
			}
		}
	}
	return nil
}

//从云上删除缓存的图像集
func (s *Scheduler) deleteCacheFaceSet(cacheList []model.PicSample, facesetName string, m *manager.Manager) error {
	if len(cacheList) > 0 {
		for _, picSample := range cacheList {
			// delete from faceset
			urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetName+model.CACHE_SUFFIX] + "/" + picSample.Id
			_, err := m.AiCloud.DeleteFace(urlStr, http.MethodDelete)
			if err != nil {
				glog.Error(err)
				return err
			}

			//delete from os
			imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + facesetName + "/" + picSample.Id
			glog.Infof("image location:%s", strings.Split(imageurl, ":"+m.CustConfig.Port)[1])
			imageaddress := m.CustConfig.StaticDir + strings.Split(imageurl, ":"+m.CustConfig.Port)[1]
			e := os.Remove(imageaddress)
			if e != nil {
				glog.Error("remove err:%s", e)
			}
		}
	}
	return nil
}

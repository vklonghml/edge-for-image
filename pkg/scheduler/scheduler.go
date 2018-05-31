package scheduler

import (
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"encoding/base64"
	"encoding/json"
	"github.com/golang/glog"
	"os"
	"strings"
)

type Scheduler struct {
}

//如果超时图片，则自动注册
func isTimeOut(pic *model.PicSample, m *manager.Manager) bool {
	return pic.UploadTime-m.LastSaveMap[pic.MostSimilarId] > m.CustConfig.PicWaitSec*1000
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

	srIdMap := model.SRIdMap{}
	srMap := model.SRDoubleMap{}

	for _, srList := range *matrix {
		for _, sr := range srList {
			if v1, ok := srIdMap[sr.From.Id]; ok {
				if sr.Similary >= v1 {
					if v2, ok := srIdMap[sr.To.Id]; ok {
						if sr.Similary >= v2 {
							//eg. a-b-86,c-d-87,当出现a-c-98时，需要删除a-b-86和c-d-87，并添加a-c-98
							for k1 := range srMap {
								for k2 := range srMap[k1] {
									if k1.Id == sr.From.Id || k1.Id == sr.To.Id || k2.Id == sr.From.Id || k2.Id == sr.To.Id {
										srMap[k1] = nil
									}
								}
							}
							srIdMap[sr.From.Id] = sr.Similary
							srIdMap[sr.To.Id] = sr.Similary
							tempToMap := model.SRSingleMap{}
							tempToMap[sr.To] = sr.Similary
							srMap[sr.From] = tempToMap
						} else {
							//eg. a-b-98，c-d-85，当出现d-a-86时，需要将c-d-85删掉
							for k1 := range srMap {
								for k2 := range srMap[k1] {
									if k1.Id == sr.From.Id || k2.Id == sr.From.Id {
										srMap[k1] = nil
									}
								}
							}
						}
					} else {
						//eg. a-b-86，当出现a-c-98时，需要删掉a-b-86，添加a-c-98
						for k1 := range srMap {
							for k2 := range srMap[k1] {
								if k1.Id == sr.From.Id || k1.Id == sr.To.Id || k2.Id == sr.From.Id || k2.Id == sr.To.Id {
									srMap[k1] = nil
								}
							}
						}
						srIdMap[sr.From.Id] = sr.Similary
						srIdMap[sr.To.Id] = sr.Similary
						tempToMap := model.SRSingleMap{}
						tempToMap[sr.To] = sr.Similary
						srMap[sr.From] = tempToMap
					}
				} else {
					//eg. a-b-98，c-d-85，当出现a-c-86时，需要删除c-d-85，保留a-b-98
					for k1 := range srMap {
						for k2 := range srMap[k1] {
							if k1.Id == sr.To.Id || k2.Id == sr.To.Id {
								srMap[k1] = nil
							}
						}
					}
				}
			} else {
				if v2, ok := srIdMap[sr.To.Id]; ok {
					if sr.Similary >= v2 {
						//eg. a-b-86，当出现d-b-87时，需要删除a-b-86，添加d-b-87
						for k1 := range srMap {
							for k2 := range srMap[k1] {
								if k1.Id == sr.From.Id || k1.Id == sr.To.Id || k2.Id == sr.From.Id || k2.Id == sr.To.Id {
									srMap[k1] = nil
								}
							}
						}
						srIdMap[sr.From.Id] = sr.Similary
						srIdMap[sr.To.Id] = sr.Similary
						tempToMap := model.SRSingleMap{}
						tempToMap[sr.To] = sr.Similary
						srMap[sr.From] = tempToMap
					}
				} else {
					//eg. a-b-98，出现c-d-85时，添加c-d-84
					srIdMap[sr.From.Id] = sr.Similary
					srIdMap[sr.To.Id] = sr.Similary
					tempToMap := model.SRSingleMap{}
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
				resp, err := m.AiCloud.AddFace(facesetName, imageurl)
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
			_, err := m.AiCloud.DeleteFace(facesetName, picSample.Id)
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

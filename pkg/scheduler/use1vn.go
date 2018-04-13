package scheduler

import (
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"github.com/golang/glog"
	"sync"
)

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
	m.DeleteFaceset(getCacheFacesetName(facesetname))
	m.CreateFacesetIfNotExist(getCacheFacesetName(facesetname))
	//s.deleteCacheFaceSet(tempRegisterCache, facesetname, m)

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

				if fromPic.UploadTime-m.LastSaveMap[toPic.MostSimilarId] > 30*1000 {
					m.SaveToDetectDB(fromPic, facesetname)

					m.LastSaveMap[toPic.MostSimilarId] = fromPic.UploadTime
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

package manager

import (
	"edge-for-image/pkg/model"
	"github.com/golang/glog"
	"sync"
)

func (m *Manager) CacheSchedulerAll() {
	//对所有的人脸集进行调度
	for k, _ := range m.FacesetMap {
		if k != "" {
			m.cacheScheduler(k);
		}
	}
}

func (m *Manager) cacheScheduler(facesetname string) {
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
		m.addCacheFaceSet(tempRegisterCache, facesetname)
		for {
			glog.Infof("for loop: tempRegisterCache size is %d.", len(tempRegisterCache))
			if len(tempRegisterCache) == 0 {
				break
			}

			picSample := &tempRegisterCache[len(tempRegisterCache)-1]
			tempRegisterCache = tempRegisterCache[:len(tempRegisterCache)-1]
			similaryRelations := m.caculateSimilarityWithCache(picSample, tempRegisterCache, picSample.ImageBase64, facesetname+model.CACHE_SUFFIX)

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
	m.deleteCacheFaceSet(tempRegisterCache, facesetname)

	//2.识别缓存
	lock.Lock()
	portal = m.UploadPortal[facesetname]
	tempDetectCache := portal.TempDetectCache
	tempDetectCache = append(tempDetectCache, portal.DetectCache...)
	portal.DetectCache = nil
	m.UploadPortal[facesetname] = portal
	lock.Unlock()

	if len(tempDetectCache) > 0 {
		m.addCacheFaceSet(tempDetectCache, facesetname)

		srMetrix := model.SRMatrix{}
		lastSaveMap := make(model.LastSaveMap) //全局？

		for _, picSample := range tempDetectCache {
			similaryRelations := m.caculateSimilarityWithCache(&picSample, tempDetectCache, picSample.ImageBase64, facesetname+model.CACHE_SUFFIX)
			srMetrix = append(srMetrix, similaryRelations)
		}

		if len(srMetrix) > 0 {
			mostSimiliarityRelations := m.caculateMostSimilarity(&srMetrix)

			tempDetectCache = nil

			for i := 0; i < len(mostSimiliarityRelations); i = i + 1 {
				fromPic := mostSimiliarityRelations[i].From
				toPic := mostSimiliarityRelations[i].To

				if fromPic.UploadTime-lastSaveMap[toPic] > 30*1000 {
					m.saveToDetectDB(fromPic, facesetname)

					lastSaveMap[toPic] = fromPic.UploadTime
				} else {
					tempDetectCache = append(tempDetectCache, *fromPic)
				}
			}
		}
		m.deleteCacheFaceSet(tempDetectCache, facesetname)
	}
}

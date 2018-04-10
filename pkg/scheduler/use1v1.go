package scheduler

import (
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"github.com/golang/glog"
	"strconv"
	"sync"
)

func (s *Scheduler) cacheSchedulerUse1v1(facesetname string, m *manager.Manager) {
	//分两个任务调度
	var lock sync.RWMutex

	//1.注册缓存
	lock.Lock()
	//将全局的Cache临时拷贝出来，进行处理
	tempRegisterCache := m.RegistCache[facesetname]
	m.RegistCache[facesetname] = nil
	glog.Infof("schedule register faceset: %s, its cache size is %d", facesetname, len(tempRegisterCache))
	lock.Unlock()

	if len(tempRegisterCache) == 1 {
		registerToDbOrBackToCache(&tempRegisterCache[0], m, facesetname)
	} else if len(tempRegisterCache) >= 2 { //more than one pic in cache
		srMatrix := s.caculateSimilarityWithCache1v1(tempRegisterCache, m)
		srList := s.caculateMostSimilarity(&srMatrix)
		for _, v := range srList {
			registerToDbOrBackToCache(v.From, m, facesetname)
		}
	}

	//2.识别缓存
	lock.Lock()
	tempDetectCache := m.DetectCache[facesetname]
	m.DetectCache[facesetname] = nil
	glog.Infof("schedule detect faceset: %s, its cache size is %d", facesetname, len(tempDetectCache))
	lock.Unlock()

	if len(tempDetectCache) == 1 {
		detectToDbOrBackToCache(&tempDetectCache[0], m, facesetname)
	} else if len(tempDetectCache) >= 2 {
		srMatrix := s.caculateSimilarityWithCache1v1(tempDetectCache, m)
		srList := s.caculateMostSimilarity(&srMatrix)
		for _, v := range srList {
			detectToDbOrBackToCache(v.From, m, facesetname)
		}
	}
}

//放入已识别库或者放回DetectCache（时间没到的话）
func detectToDbOrBackToCache(pic *model.PicSample, m *manager.Manager, facesetname string) {
	if isTimeOut(pic.UploadTime) { //if timeout, the detect
		m.SaveToDetectDB(pic, facesetname)
	} else { //if not timeout, then put back to DetectCache
		m.DetectCache[facesetname] = append(m.DetectCache[facesetname], *pic)
	}
}

//注册或者放回RegisterCache（时间没到的话）
func registerToDbOrBackToCache(pic *model.PicSample, m *manager.Manager, facesetname string) {
	if isTimeOut(pic.UploadTime) { //if timeout, the register
		m.SaveToRegisterDB(pic, facesetname)
	} else { //if not timeout, then put back to RegistCache
		m.RegistCache[facesetname] = append(m.RegistCache[facesetname], *pic)
	}
}

//计算相似度矩阵，计算次数为 n * (n-1) / 2.
func (scheduler *Scheduler) caculateSimilarityWithCache1v1(cacheList []model.PicSample, m *manager.Manager) (matrix model.SRMatrix) {
	l := len(cacheList)
	for i := l - 1; i > 0; i-- {
		list := model.SRList{}
		for j := i - 1; j >= 0; j-- {
			sr := caculateSimilarityWithOther(&cacheList[i], &cacheList[j], m)
			if sr.Similary > int32(m.CustConfig.Similarity) {
				list = append(list, sr)
			}
		}
		matrix = append(matrix, list)
	}
	return matrix
}

//计算两张图片的相似度，返回SimilaryRelation
func caculateSimilarityWithOther(pic1 *model.PicSample, pic2 *model.PicSample, m *manager.Manager) (sr model.SimilaryRelation) {
	resp := m.FaceVerify(pic1.ImageUrl, pic2.ImageUrl)
	sr.From = pic1
	sr.To = pic2
	f, _ := strconv.ParseFloat(resp.Similarity, 32)
	sr.Similary = int32(f * 100)
	return sr
}

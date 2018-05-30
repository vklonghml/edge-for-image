package scheduler

import (
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"github.com/golang/glog"
	"strconv"
	"sync"
	"errors"
)

func (s *Scheduler) ScheduleRegisterCacheUse1v1(facesetname string, m *manager.Manager) {
	//分两个任务调度
	var lock sync.RWMutex

	//1.注册缓存
	lock.Lock()
	//将全局的Cache临时拷贝出来，进行处理
	var tempRegisterCache []model.PicSample
	if len(m.RegistCache[facesetname]) <= m.CustConfig.RegistCacheSize {
		tempRegisterCache = m.RegistCache[facesetname]
		m.RegistCache[facesetname] = nil
	} else {
		tempRegisterCache = m.RegistCache[facesetname][:m.CustConfig.RegistCacheSize]
		m.RegistCache[facesetname] = m.RegistCache[facesetname][m.CustConfig.RegistCacheSize:]
	}
	glog.Infof("schedule register faceset: %s, its cache size is %d", facesetname, len(tempRegisterCache))
	lock.Unlock()

	if !m.CloseToRegist[facesetname] {
		glog.Infof("pujie1")
		count, err := m.CountRegisterDB(facesetname)
		if err != nil {
			glog.Errorf("count faceset : %s , err: %s", facesetname, err.Error())
		}

		if count >= m.CustConfig.AutoRegistSize {
			m.CloseToRegist[facesetname] = true
		}
	}

	if len(tempRegisterCache) == 1 {
		glog.Infof("temp register cache size is 1, faceset name is %v.", facesetname)
		if !m.CloseToRegist[facesetname] {
			registerToDb(&tempRegisterCache[0], m, facesetname)
		} else {
			unknowToDb(&tempRegisterCache[0], m, facesetname)
		}

	} else if len(tempRegisterCache) >= 2 { //more than one pic in cache
		srMatrix, remains := s.caculateSimilarityWithCache1v1(tempRegisterCache, m)
		glog.Infof("temp register cache size is %v, faceset name is %v.", len(tempRegisterCache) ,facesetname)
		srList := s.caculateMostSimilarity(&srMatrix)

		for _, v := range srList {
			if !m.CloseToRegist[facesetname] {
				registerToDb(v.From, m, facesetname)
			} else {
				unknowToDb(v.From, m, facesetname)
			}
		}

		for i := range remains {
			if !m.CloseToRegist[facesetname] {
				registerToDb(&remains[i], m, facesetname)
			} else {
				unknowToDb(&remains[i], m, facesetname)
			}
		}
	}
}

func (s *Scheduler) ScheduleDetectCacheUse1v1(facesetname string, m *manager.Manager) {
	var lock sync.RWMutex
	//2.识别缓存
	lock.Lock()
	var tempDetectCache []model.PicSample
	if len(m.DetectCache[facesetname]) < m.CustConfig.DetectCacheSize {
		tempDetectCache = m.DetectCache[facesetname]
		m.DetectCache[facesetname] = nil
	} else {
		tempDetectCache = m.DetectCache[facesetname][:m.CustConfig.DetectCacheSize]
		m.DetectCache[facesetname] = m.DetectCache[facesetname][m.CustConfig.DetectCacheSize:]
	}
	glog.Infof("schedule detect faceset: %s, its cache size is %d", facesetname, len(tempDetectCache))
	lock.Unlock()

	if len(tempDetectCache) == 1 {
		detectToDb(&tempDetectCache[0], m, facesetname)
	} else if len(tempDetectCache) >= 2 {
		srMatrix, remains := s.caculateSimilarityWithCache1v1(tempDetectCache, m)
		srList := s.caculateMostSimilarity(&srMatrix)
		for _, v := range srList {
			detectToDb(v.From, m, facesetname)
		}

		for _, v := range remains {
			detectToDb(&v, m, facesetname)
		}
	}
}

//放入已识别库或者放回DetectCache（时间没到的话）
func detectToDb(pic *model.PicSample, m *manager.Manager, facesetname string) {
	if pic.ImageUrl == "" {
		return
	}

	if isTimeOut(pic, m) { //if timeout, the detect
		m.SaveToDetectDB(pic, facesetname)
		m.LastSaveMap[pic.MostSimilarId] = pic.UploadTime
	} else {
		glog.Infof("the image has been detect recent, ignored this detect, url is %v.", pic.ImageUrl)
	}
}

//注册或者放回RegisterCache（时间没到的话）
func registerToDb(pic *model.PicSample, m *manager.Manager, facesetname string) {
	if pic.ImageUrl == "" {
		return
	}

	if !m.RingBuffer[facesetname].IsEmpty() {
		glog.Infof("ringbuffer cap is %d, length is %d.", m.RingBuffer[facesetname].Capacity, m.RingBuffer[facesetname].Length)
		for _, e := range m.RingBuffer[facesetname].Data {
			pic0, ok := e.(*model.PicSample)
			if !ok {
				glog.Errorf("the e in ringBuffer is nil, now continue.")
				continue
			}
			sr, err := caculateSimilarityWithOther(pic0, pic, m)
			if err != nil {
				glog.Error("caculate ringbuffer similary failed.")
				return
			}
			if sr.Similary > int32(m.CustConfig.Similarity) {
				glog.Infof("%s has been registed, here not save to DB, return.", pic.Id)
				return
			}
		}
	}

	m.CaculateSimilarity(pic, pic.ImageBase64, facesetname)
	m.CaculateMostSimilarity(pic)

	if pic.MostSimilar < int32(m.CustConfig.Similarity) {
		m.SaveToRegisterDB(pic, facesetname)
	} else {
		glog.Infof("pic has been registed.")
	}
}

func unknowToDb(pic *model.PicSample, m *manager.Manager, facesetname string) {
	if pic.ImageUrl == "" {
		return
	}

	if !m.RingBuffer[facesetname].IsEmpty() {
		m.RingBuffer[facesetname].Each(func(node interface{}) {
			sr, err := caculateSimilarityWithOther(node.(*model.PicSample), pic, m)
			if err != nil {
				glog.Error("caculate ringbuffer similary failed.")
				return
			}
			if sr.Similary > int32(m.CustConfig.Similarity) {
				glog.Error("pic has been registed.")
			}
		})
	}

	m.CaculateSimilarity(pic, pic.ImageBase64, facesetname)
	m.CaculateMostSimilarity(pic)

	if pic.MostSimilar < int32(m.CustConfig.Similarity) {
		m.SaveToRegisterDB(pic, facesetname)
	} else {
		glog.Infof("pic has been registed.")
	}
}

//计算相似度矩阵，计算次数为 n * (n-1) / 2.
func (scheduler *Scheduler) caculateSimilarityWithCache1v1(cacheList []model.PicSample, m *manager.Manager) (matrix model.SRMatrix, isolutes []model.PicSample) {

	resultMatrix := model.SRMatrix{}
	resultIsolutes := []model.PicSample{}
	isolutedMap := make(map[*model.PicSample]int32)

	l := len(cacheList)
	for i := l - 1; i > 0; i-- {
		list := model.SRList{}
		for j := i - 1; j >= 0; j-- {
			sr, err := caculateSimilarityWithOther(&cacheList[i], &cacheList[j], m)
			if err != nil {
				continue
			}

			if sr.Similary > isolutedMap[&cacheList[i]] {
				isolutedMap[&cacheList[i]] = sr.Similary
			}

			if sr.Similary > isolutedMap[&cacheList[j]] {
				isolutedMap[&cacheList[j]] = sr.Similary
			}

			if sr.Similary > int32(m.CustConfig.Similarity) {
				list = append(list, sr)
			}
		}
		resultMatrix = append(resultMatrix, list)
	}

	for k, v := range isolutedMap {
		if k.ImageUrl != "" && v < int32(m.CustConfig.Similarity) {
			resultIsolutes = append(resultIsolutes, *k)
		}
	}

	return resultMatrix, resultIsolutes
}

//计算两张图片的相似度，返回SimilaryRelation
func caculateSimilarityWithOther(pic1 *model.PicSample, pic2 *model.PicSample, m *manager.Manager) (sr model.SimilaryRelation, err error) {

	if pic1.ImageUrl == "" {
		return sr, errors.New("pic image url is nil.")
	}

	if pic2.ImageUrl == "" {
		return sr, errors.New("pic image url is nil.")
	}

	resp := m.FaceVerify(pic1.ImageUrl, pic2.ImageUrl)
	sr.From = pic1
	sr.To = pic2
	f, _ := strconv.ParseFloat(resp.Similarity, 32)
	sr.Similary = int32(f * 100)
	return sr, nil
}

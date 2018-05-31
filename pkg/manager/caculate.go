package manager

import (
	"edge-for-image/pkg/model"
	"fmt"
	"github.com/golang/glog"
	"strconv"
)

func (m *Manager) CaculateSimilarity(picSample *model.PicSample, facesetname string) error {
	resp, err := m.AiCloud.FaceSearch(facesetname, picSample.ImageUrl)
	if err != nil {
		glog.Errorf(err.Error())
		return err
	}

	if len(resp.Faces) == 0 {
		glog.Errorf("search image from faceset %s return 0 face.", facesetname)
		return nil
	}

	for _, v := range resp.Faces {
		faceid := v.FaceId
		similar := v.Similarity * 100
		picSample.Similarity[faceid] = int32(similar)
		glog.Infof("face similarity: %f", v.Similarity)
	}

	return nil
}

//计算picsample与注册库中最相似的id
func (m *Manager) CaculateMostSimilarity(sample *model.PicSample) {
	var curMostSimilarityValue int32 = 0
	curMostSimilarityKey := ""
	if len(sample.Similarity) > 0 {
		for k, v := range sample.Similarity {
			if v > curMostSimilarityValue {
				curMostSimilarityKey = k
				curMostSimilarityValue = v
			}
		}
		sample.MostSimilar = curMostSimilarityValue
		value, _ := strconv.ParseInt(curMostSimilarityKey, 10, 32)

		sample.MostSimilarId = fmt.Sprintf("%d", value)
		glog.Infof("CaculateMostSimilarity: faceid is %s.", sample.MostSimilarId)
	}
}

package manager

import (
	"bytes"
	"edge-for-image/pkg/model"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"os"
	"strconv"
)

func (m *Manager) CaculateSimilarity(picSample *model.PicSample, imageBase64 string, facesetname string) error {

	//base64解码
	imageDecode, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		glog.Error("DecodeString err")
		return errors.New("DecodeString image err")
	}
	buf := bytes.NewBuffer(nil)
	imageReader := bytes.NewReader(imageDecode)
	if _, err := io.Copy(buf, imageReader); err != nil {
		glog.Errorf("file copy to buf err: %s", err.Error())
	}

	// first detect image face
	jdface, err := m.DetectFace(imageBase64)
	if err != nil {
		glog.Error(err)
		return err
	}
	//这个日志不可读，打印出来没意义。
	//glog.Infof("dface: %#v", jdface)

	// search face in faceset
	// /v1/faceSet/13345/faceSearch?url=http://100.114.203.102/data/2_8.png
	//如果检测到人脸了，就先存储到文件系统，然后进行1：N的人脸搜索
	if jdface != nil {

		//这里的sample.Id 就是图片在文件系统中存储的名字，格式为 UUID + 上传的原始图片名.
		imageaddress := m.CustConfig.StaticDir + "/" + facesetname + "/" + picSample.Id
		glog.Infof("the image file path is %s", imageaddress)
		fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			glog.Error(err)
			return err
		}
		defer fileToSave.Close()
		if _, err := io.Copy(fileToSave, buf); err != nil {
			glog.Errorf("buf copy to file err: %s", err.Error())
		}

		//进行1：N的搜索
		imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + facesetname + "/" + picSample.Id
		picSample.ImageAddress = imageaddress
		picSample.ImageUrl = imageurl
		picSample.Face = jdface
		resp, err := m.AiCloud.FaceSearch(facesetname, imageurl)
		if err != nil {
			glog.Errorf(err.Error())
			return err
		}

		if len(resp.Faces) == 0 {
			glog.Errorf("search image from faceset %s failed.", facesetname)
			return nil
		} else if len(resp.Faces) > 0 {
			// found := false
			for _, v := range resp.Faces {
				faceid := v.FaceId
				similar := v.Similarity * 100
				picSample.Similarity[faceid] = int32(similar)
				glog.Infof("face similarity: %f", v.Similarity)
			}
		}
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

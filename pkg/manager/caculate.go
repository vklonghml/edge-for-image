package manager

import (
	"bytes"
	"edge-for-image/pkg/model"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		glog.Infof("the imageaddress is %s", imageaddress)
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
		urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetname] + "/faceSearch?url=" + imageurl
		// body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\"}", imageurl))
		// resp, err := m.AiCloud.FakeFaceSearch(urlStr, http.MethodPost, body)
		resp, err := m.AiCloud.FaceSearch(urlStr, http.MethodGet, nil)
		if err != nil {
			glog.Errorf(err.Error())
			return err
		}
		if strings.Contains(string(resp), "have no face") {
			glog.Error("image have no face")
			return nil
		}
		data := resp
		// glog.Infof("resp :%#v", data)
		bdata := make(map[string]interface{})
		err = json.Unmarshal(data, &bdata)
		if err != nil {
			glog.Errorf(err.Error())
			return err
		}
		faces := bdata["faces"].([]interface{})

		// glog.Infof("resp :%#v", faces)
		//largeface := make(map[string]interface{})
		//var largesimilar int64
		if len(faces) > 0 {
			// found := false
			for _, v := range faces {
				face := v.(map[string]interface{})
				faceid := face["faceID"].(string)
				similar, err := strconv.ParseInt(face["similarity"].(string), 10, 32)
				if err != nil {
					glog.Errorf("parse error: %s", err.Error())
				}
				picSample.Similarity[faceid] = int32(similar)
				glog.Infof("face similarity: %s", face["similarity"])
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

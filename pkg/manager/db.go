package manager

import (
	"edge-for-image/pkg/db"
	"edge-for-image/pkg/model"
	"fmt"
	"github.com/golang/glog"
	"time"
)

//保存到facedb数据库
func (m *Manager) SaveToRegisterDB(picSample *model.PicSample, facesetName string) error {
	glog.Infof("SaveToRegisterDB: picSampleUrl is %s.", picSample.ImageUrl)
	resp := m.AddFaceToSet(picSample.ImageUrl, facesetName)
	err := db.InsertIntoFacedb(m.Mydb, facesetName, resp.FaceID, nil, "", "", "", "", picSample.ImageAddress, picSample.ImageUrl, time.Now().UnixNano()/1e6, "", "", "facedb")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
	return err
}

//保存到已识别的knowfaceinfo数据库
func (m *Manager) SaveToDetectDB(picSample *model.PicSample, facesetName string) error {
	glog.Infof("SaveToDetectDB: picSampleUrl is %s.", picSample.ImageUrl)
	err := m.insertIntoKnow(fmt.Sprintf("%d", picSample.MostSimilar), picSample.ImageAddress, picSample.ImageUrl, picSample.MostSimilarId, facesetName)
	//err := db.InsertIntoFacedb(m.Mydb, facesetName, picSample.Id, picSample.Face, "", "", "", "", picSample.ImageAddress, picSample.ImageUrl, time.Now().UnixNano()/1e6, picSample.MostSimilarUrl, string(picSample.MostSimilar), "knowfaceinfo")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
	return err
}

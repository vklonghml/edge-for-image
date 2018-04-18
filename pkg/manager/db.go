package manager

import (
	"edge-for-image/pkg/db"
	"edge-for-image/pkg/model"
	"fmt"
	"github.com/golang/glog"
)

//保存到facedb数据库
func (m *Manager) SaveToRegisterDB(picSample *model.PicSample, facesetName string) error {

	if picSample.ImageUrl == "" {
		return nil
	}


	glog.Infof("SaveToRegisterDB: picSampleUrl is %s.", picSample.ImageUrl)
	resp := m.AddFaceToSet(picSample.ImageUrl, facesetName)
	err := db.InsertIntoFacedb(m.Mydb, facesetName, resp.FaceID, nil, "", resp.FaceID, "", "", picSample.ImageAddress, picSample.ImageUrl, picSample.UploadTime, "", "", "facedb")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}

	if m.RingBuffer.IsFull() {
		m.RingBuffer.OutElement()
		m.RingBuffer.Append(picSample)
	} else {
		m.RingBuffer.Append(picSample)
	}

	return err
}

//保存到unknowfaceinfo数据库
func (m *Manager) SaveToUnknowDB(picSample *model.PicSample, facesetName string) error {

	if picSample.ImageUrl == "" {
		return nil
	}

	glog.Infof("SaveToUnknowDB: picSampleUrl is %s.", picSample.ImageUrl)
	resp := m.AddFaceToSet(picSample.ImageUrl, facesetName)
	err := db.InsertIntoFacedb(m.Mydb, facesetName, resp.FaceID, nil, "", resp.FaceID, "", "", picSample.ImageAddress, picSample.ImageUrl, picSample.UploadTime, "", "", "unknowfaceinfo")
	if err != nil {
		glog.Errorf("Prepare INSERT unknowfaceinfo err: %s", err.Error())
	}

	if m.RingBuffer.IsFull() {
		m.RingBuffer.OutElement()
		m.RingBuffer.Append(picSample)
	} else {
		m.RingBuffer.Append(picSample)
	}

	return err
}

//保存到已识别的knowfaceinfo数据库
func (m *Manager) SaveToDetectDB(picSample *model.PicSample, facesetName string) error {
	if picSample.ImageUrl == "" {
		return nil
	}

	glog.Infof("SaveToDetectDB: picSampleUrl is %s.", picSample.ImageUrl)
	err := m.insertIntoKnow(fmt.Sprintf("%d", picSample.MostSimilar), picSample.ImageAddress, picSample.ImageUrl, picSample.MostSimilarId, facesetName)
	//err := db.InsertIntoFacedb(m.Mydb, facesetName, picSample.Id, picSample.Face, "", "", "", "", picSample.ImageAddress, picSample.ImageUrl, time.Now().UnixNano()/1e6, picSample.MostSimilarUrl, string(picSample.MostSimilar), "knowfaceinfo")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
	return err
}

func (m *Manager) CountRegisterDB(facesetname string) (int, error) {
	return m.countFacedb(facesetname)
}

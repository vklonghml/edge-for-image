package manager

import (
	"edge-for-image/pkg/db"
	"edge-for-image/pkg/model"
	"encoding/json"
	"errors"
	"io/ioutil"
	// "path/filepath"
	// "flag"
	"fmt"

	"database/sql"
	"os"
	"strconv"

	// "mime/multipart"
	// "regexp"
	"strings"
	"time"

	"edge-for-image/pkg"
	"edge-for-image/pkg/accessai"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/satori/go.uuid"
	"edge-for-image/pkg/obs"
)

var FACE_SET_CAPACITY int64 = 100000

type Manager struct {
	CustConfig    *pkg.Config
	AiCloud       *accessai.Accessai
	ObsClient     *obs.ObsClient
	Mydb          *sql.DB
	IAMClient     *accessai.IAMClient
	FaceidMap     map[string]string
	UploadPortal  map[string]model.Caches //to delete
	DetectCache   map[string][]model.PicSample
	RegistCache   map[string][]model.PicSample
	LastSaveMap   map[string]int64
	CloseToRegist map[string]bool
	RingBuffer    map[string]*model.Queen
	RegistThread  map[string]int32
	DetectThread  map[string]int32
}

type BoundingBox struct {
	TopLeftX string `json:"topleftx"`
	TopLeftY string `json:"toplefty"`
	Width    string `json:"width"`
	Height   string `json:"height"`
}

type Face struct {
	Confidence string      `json:"confidence"`
	Bound      BoundingBox `json:"bound"`
}

type FaceSet struct {
	FaceSetName string `json:"facesetname"`
	FaceSetID   string `json:"facesetid"`
	CreateTime  string `json:"createtime"`
}

type FaceInfo struct {
	Id           string
	FaceSetName  string `json:"facesetname"`
	FaceID       string `json:"faceid"`
	Face         Face   `json:"face"`
	ImageBase64  string `json:"imagebase64"`
	Name         string `json:"name"`
	Age          string `json:"age"`
	Address      string `json:"address"`
	Imageaddress string `json:"imageaddress"`
	ImageURL     string `json:"imageurl"`
	// alreadyknow bool    `json:"alreadyknow"`
	CreateTime       int64  `json:"createtime"`
	SimilaryImageURL string `json:"similaryimageURL"`
	Similarity       string `json:"similarity"`
}

func (m *Manager) listAlllfiles(facesetname, filename string) bool {
	files, err := ioutil.ReadDir(m.CustConfig.StaticDir + "/" + facesetname)
	if err != nil {
		glog.Error(err)
	}
	for _, files := range files {
		if files.Name() == filename {
			return true
		}
	}
	return false
}

func (m *Manager) DetectFace(imagebase64 string) ([]byte, error) {
	// first detect image face
	resp, err := m.AiCloud.FaceDetectBase64(imagebase64)
	if err != nil {
		glog.Errorf(err.Error())
		return nil, err
	}

	if len(resp.Faces) == 0 {
		glog.Errorf("detect face number is 0")
		return nil, errors.New("dfaces len is 0")
	}

	jdface, err := json.Marshal(resp.Faces[0])
	if err != nil {
		glog.Errorf("Marshal face err: %s", err)
		return nil, err
	}
	return jdface, nil
}

func (m *Manager) CreateFacesetIfNotExist(facesetname string) error {
	// rows, err := db.Query("select * from faceset where facesetname = ?", config.FaceSetName)
	for key := range m.FaceidMap {
		if key == facesetname {
			return nil
		}
	}
	glog.Infof("faceset %s is not exist, now creating faceset", facesetname)

	resp, err := m.AiCloud.CreateFaceset(facesetname, FACE_SET_CAPACITY)
	if err != nil {
		glog.Errorf("create faceset err: %s", err.Error())
		return err
	}

	stmt, err := m.Mydb.Prepare("INSERT faceset SET facesetname=?,facesetid=?,createtime=?")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
		return err
	}
	defer stmt.Close()
	//glog.Infof("config faceset name=%s", facesetname)
	_, err = stmt.Exec(facesetname, resp.FaceSetInfo.FaceSetId, time.Now().UnixNano()/1e6)
	if err != nil {
		glog.Errorf("INSERT faceinfo err: %s", err.Error())
		return err
	}
	m.FaceidMap[facesetname] = resp.FaceSetInfo.FaceSetId

	direc := m.CustConfig.StaticDir + "/" + facesetname
	if _, err := os.Stat(direc); os.IsNotExist(err) {
		err = os.Mkdir(direc, 0777)
		if err != nil {
			glog.Errorf("mkdir dir err: %s", err.Error())
			return err
		}
	}
	glog.Infof("create facesetname: %s", facesetname)
	return nil
}

func (m *Manager) searchFace(imageBase64, imagename, facesetname string) error {
	if err := m.CreateFacesetIfNotExist(facesetname); err != nil {
		return err
	}

	if m.RegistThread[facesetname] == 0 {
		m.RegistThread[facesetname] = 1
	}
	if m.DetectThread[facesetname] == 0 {
		m.DetectThread[facesetname] = 1
	}

	uid, err := uuid.NewV4()
	if err != nil {
		glog.Errorf("uuid generate error: %s", err.Error())
	}
	imagename = fmt.Sprintf("%s", uid) + "-" + imagename
	glog.Infof("the imagename is %s", imagename)

	inputKey := facesetname + "/" + imagename
	err = m.UploadImageToObs(inputKey, imageBase64)
	if err != nil {
		glog.Errorf("Upload Image %s to OBS failed!", inputKey)
		return err
	}

	//构造PicSample对象
	picSample := &model.PicSample{
		Id:          imagename,
		UploadTime:  time.Now().UnixNano() / 1e6,
		Similarity:  make(map[string]int32),
		MostSimilar: 0,
		ImageUrl:    inputKey,
	}

	//调用人脸比对API计算相似度
	m.CaculateSimilarity(picSample, facesetname)
	m.CaculateMostSimilarity(picSample)

	if picSample.MostSimilar > int32(m.CustConfig.Similarity) {
		//m.DetectCache[facesetname] = append(m.DetectCache[facesetname], *picSample)
		m.SaveToDetectDB(picSample, facesetname)
	} else {
		//m.RegistCache[facesetname] = append(m.RegistCache[facesetname], *picSample)
	}
	glog.Infof("====================================================== end here   ====================")
	return nil
}

func (m *Manager) insertIntoKnow(largesimilar, imageaddress, imageurl string, faceid string, facesetname string) error {
	rows, err := m.Mydb.Query("select * from facedb where faceid = ?", faceid)
	if err != nil {
		glog.Errorf("Query db err: %s", err.Error())
		return err
	}
	defer rows.Close()
	var knowface FaceInfo
	// var faceinteface interface{}
	faceinteface := make([]byte, 255)
	var id int
	knowsfaces := make([]FaceInfo, 0)
	for rows.Next() {
		err := rows.Scan(&id, &knowface.FaceSetName, &knowface.FaceID, &faceinteface, &knowface.ImageBase64, &knowface.Name, &knowface.Age, &knowface.Address, &knowface.Imageaddress, &knowface.ImageURL, &knowface.CreateTime, &knowface.SimilaryImageURL, &knowface.Similarity)
		if err != nil {
			glog.Errorf("scan db err: %s", err.Error())
			return err
		}
		knowsfaces = append(knowsfaces, knowface)
	}
	if len(knowsfaces) == 1 {
		// insert know face
		//glog.Infof("byte:%#v", faceinteface)
		// jsonface, err := json.Marshal(faceinteface)
		// if err != nil {
		// 	glog.Errorf("Marshal face err: %s", err.Error())
		// 	return err
		// }
		err = db.InsertIntoFacedb(m.Mydb, facesetname, strconv.Itoa(id), faceinteface, "", knowface.Name, knowface.Age, knowface.Address, imageaddress, imageurl, time.Now().UnixNano()/1e6, knowface.ImageURL, largesimilar, "knowfaceinfo")
		if err != nil {
			glog.Errorf("INSERT faceinfo err: %s", err)
			return err
		}
		//glog.Infof("found similarity face: %s", face["similarity"])
	} else if len(knowsfaces) == 0 {
		//glog.Errorf("facedb has no record of that faceid: %s", face["faceID"])
		return errors.New("facedb has no record of that faceid")
	} else {
		//glog.Errorf("facedb has two many faces that match the faceid: %s", face["faceID"])
		return errors.New("facedb has two many faces that match the faceid")
	}
	return nil
}

// func (m *Manager) insertIntoFacedb(facesetname, imageaddress, imageurl, name, age, address string, face interface{}) {
func (m *Manager) insertIntoFacedb(facesetname, imageaddress, imageurl, name, age, address string, face []byte) {
	resp, err := m.AiCloud.AddFace(facesetname, imageurl)
	if err != nil {
		glog.Errorf("add face err: %s", err.Error())
		return
	}

	if len(resp.Faces) == 0 {
		glog.Errorf("add face return 0 faces.")
		return
	}

	err = db.InsertIntoFacedb(m.Mydb, facesetname, resp.Faces[0].FaceId, face, "", name, age, address, imageaddress, imageurl, time.Now().UnixNano()/1e6, "", "", "facedb")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
}

func (m *Manager) GetAllfaces(facesetname, table string, start, end, numbers int64, timeby bool) ([]FaceInfo, error) {
	sort := "desc"
	if !timeby {
		sort = "asc"
	}
	qstr := fmt.Sprintf("select * from %s where facesetname='%s' order by createtime %s", table, facesetname, sort)
	//glog.Infof(qstr)
	rows, err := m.Mydb.Query(qstr)
	if err != nil {
		glog.Errorf("Query db err: %s", err.Error())
		return nil, err
	}
	defer rows.Close()
	var knowface FaceInfo
	// var id int
	var index int64
	var num int64
	num = 0
	index = 0
	knowsfaces := make([]FaceInfo, 0)
	faceinteface := make([]byte, 255)

	for rows.Next() && (numbers == 0 || num < numbers) {
		// var faceinteface interface{}
		err := rows.Scan(&knowface.Id, &knowface.FaceSetName, &knowface.FaceID, &faceinteface, &knowface.ImageBase64, &knowface.Name, &knowface.Age, &knowface.Address, &knowface.Imageaddress, &knowface.ImageURL, &knowface.CreateTime, &knowface.SimilaryImageURL, &knowface.Similarity)
		if err != nil {
			glog.Errorf("scan db err: %s", err.Error())
			return nil, err
		}
		index, _ = strconv.ParseInt(knowface.Id, 10, 64)
		// glog.Infof("index:%s", index)
		if !((start == -1 || (start > 0 && index >= start)) && (end == -1 || (end > 0 && index <= end))) {
			continue
		}
		// glog.Infof("numstart:%s", num)
		if faceinteface != nil && len(faceinteface) != 0 {
			values := make(map[string]interface{})
			err = json.Unmarshal(faceinteface, &values)
			if err != nil {
				glog.Error(err)
				return nil, err
			}
			// values := faceinteface.(map[string]interface{})
			if values != nil {
				knowface.Face.Confidence = values["confidence"].(string)
				if _, ok := values["boundingBox"]; ok {
					secvalue := values["boundingBox"].(map[string]interface{})
					if secvalue != nil {
						knowface.Face.Bound.Height = strconv.FormatFloat(secvalue["height"].(float64), 'f', 0, 64)
						knowface.Face.Bound.Width = strconv.FormatFloat(secvalue["width"].(float64), 'f', 0, 64)
						knowface.Face.Bound.TopLeftX = strconv.FormatFloat(secvalue["topLeftX"].(float64), 'f', 0, 64)
						knowface.Face.Bound.TopLeftY = strconv.FormatFloat(secvalue["topLeftY"].(float64), 'f', 0, 64)
					}
				}
			}
		}
		knowface.ImageURL = fmt.Sprintf("https://%s.obs.cn-north-1.myhwclouds.com/%s", pkg.Config0.OBSBucketName, knowface.ImageURL)
		knowface.SimilaryImageURL = fmt.Sprintf("https://%s.obs.cn-north-1.myhwclouds.com/%s", pkg.Config0.OBSBucketName, knowface.SimilaryImageURL)
		knowsfaces = append(knowsfaces, knowface)
		num++
	}
	return knowsfaces, nil
}

func (m *Manager) Deletefaces(facesetname string, blist []interface{}, deleteimage bool) error {
	glog.Infof("%#v", blist)
	var id string
	var faceid string
	var imageurl string
	var isknown string
	for _, value := range blist {
		v, ok := value.(map[string]interface{})
		if ok && v != nil {
			id = v["Id"].(string)
			faceid = v["faceid"].(string)
			imageurl = v["imageurl"].(string)
			isknown = v["isknown"].(string)
		} else {
			v2 := value.(map[string]string)
			id = v2["Id"]
			faceid = v2["faceid"]
			imageurl = v2["imageurl"]
			isknown = v2["isknown"]
		}
		// glog.Infof("%s, %s, %s", id, faceid, imageurl)

		if isknown == "0" {
			// glog.Infof("in delete knowfaceinfo")
			_, err := m.Mydb.Exec("delete from knowfaceinfo where id=? and faceid=? and facesetname=?", id, faceid, facesetname)
			if err != nil {
				glog.Error(err)
				return err
			}
		} else if isknown == "1" {
			// glog.Infof("in delete unknowfaceinfo")
			_, err := m.Mydb.Exec("delete from unknowfaceinfo where id=? and facesetname=?", id, facesetname)
			if err != nil {
				glog.Error(err)
				return err
			}
			// if deleteimage {
			// 	imageaddress := m.CustConfig.StaticDir+ "/" +facesetname + strings.Split(imageurl, ":"+m.CustConfig.Port)[1]
			// 	err = os.Remove(imageaddress)
			// 	if err != nil {
			// 		glog.Error("remove err:%s", err)
			// 	}
			// }
		} else if isknown == "2" {
			// delete from faceset
			_, err := m.AiCloud.DeleteFace(facesetname, faceid)
			if err != nil {
				glog.Error(err)
				return err
			}

			_, err = m.Mydb.Exec("delete from facedb where id=? and facesetname=?", id, facesetname)
			if err != nil {
				glog.Error(err)
				return err
			}
		} else {
			glog.Warning("unknow face table to delete")
		}

		if deleteimage {
			glog.Infof("image location:%s", strings.Split(imageurl, ":"+m.CustConfig.Port)[1])
			imageaddress := m.CustConfig.StaticDir + strings.Split(imageurl, ":"+m.CustConfig.Port)[1]
			err := os.Remove(imageaddress)
			if err != nil {
				glog.Error("remove err:%s", err)
			}
		}

	}
	return nil
}

func (m *Manager) updateface(face map[string]interface{}) error {
	id := face["Id"].(string)
	facesetname := face["facesetname"].(string)
	faceid := face["faceid"].(string)
	name := face["name"].(string)
	age := face["age"].(string)
	address := face["address"].(string)
	imageurl := face["imageurl"].(string)
	isadd := face["isadd"].(bool)
	isknown := face["isknown"].(string)
	facem, err := json.Marshal(face["face"])
	if err != nil {
		return err
	}
	m.CreateFacesetIfNotExist(facesetname)
	// glog.Infof("face:%#v", facem)
	if isknown == "1" && isadd && faceid == "" {
		splitstr := ":" + m.CustConfig.Port
		imageaddress := m.CustConfig.StaticDir + strings.Split(imageurl, splitstr)[1]
		m.insertIntoFacedb(facesetname, imageaddress, imageurl, name, age, address, facem)
		blist := make([]interface{}, 0)
		fmap := make(map[string]string)
		fmap["Id"] = id
		fmap["faceid"] = faceid
		fmap["imageurl"] = imageurl
		fmap["isknown"] = "1"
		blist = append(blist, fmap)
		m.Deletefaces(facesetname, blist, false)
	} else {
		// glog.Infof("faceid:%s", faceid)
		if isknown == "0" {
			_, err := m.Mydb.Exec("update knowfaceinfo set name=?, age=?, address=? where id=? and faceid=?", name, age, address, id, faceid)
			if err != nil {
				return err
			}
		} else if isknown == "1" {
			_, err := m.Mydb.Exec("update unknowfaceinfo set name=?, age=?, address=? where id=?", name, age, address, id)
			if err != nil {
				return err
			}
		} else if isknown == "2" {
			_, err := m.Mydb.Exec("update facedb set name=?, age=?, address=? where id=?", name, age, address, id)
			if err != nil {
				return err
			}
			//need update knowfaceinfo meanwhile. The imageurl of facedb is equal to the similaryimageURL of knowfaceinfo.
			_, knowErr := m.Mydb.Exec("update knowfaceinfo set name=?, age=?, address=? where similaryimageURL=?", name, age, address, imageurl)
			if knowErr != nil {
				return knowErr
			}
		} else {
			glog.Warning("unknow table to update")
		}
	}
	return nil
}

func (m *Manager) countFacedb(facesetname string) (int, error) {
	count := 0
	rows, err := m.Mydb.Query("select id from facedb where facesetname=?", facesetname)
	if err != nil {
		return 0, err
	}

	defer rows.Close()
	for rows.Next() {
		count = count + 1
	}
	return count, nil
}

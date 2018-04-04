package main

import (
	// "bufio"
	"bytes"
	"container/list"
	"edge-for-image/pkg/model"
	"edge-for-image/pkg/payload"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"sync"

	// "path/filepath"
	// "flag"
	"fmt"
	// "log"
	"crypto/md5"
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
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

	"github.com/robfig/cron"
)

// var (
// 	CustConfig *pkg.Config
// 	AiCloud    *accessai.Accessai
// 	Mydb       *sql.DB
// )

// func init() {
// 	AiCloud = accessai.NewAccessai()
// 	CustConfig = pkg.InitConfig()
// }

// Config use for server
// type Config struct {
// 	port             string
// 	staticDir        string
// 	aiurl            string
// }

type Manager struct {
	CustConfig   *pkg.Config
	AiCloud      *accessai.Accessai
	Mydb         *sql.DB
	FaceidMap    map[string]string
	UploadPortal *model.UploadPortal
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

func checkFacesetExist(config *pkg.Config, aicloud *accessai.Accessai, facesetmap map[string]string) *sql.DB {
	db, err := sql.Open("mysql", config.DBConnStr)
	if err != nil {
		glog.Errorf("open db err: %s", err.Error())
		os.Exit(1)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		glog.Errorf("ping db err: %s", err.Error())
		os.Exit(1)
	}
	// rows, err := db.Query("select * from faceset where facesetname = ?", config.FaceSetName)
	rows, err := db.Query("select * from faceset")
	if err != nil {
		db.Close()
		glog.Errorf("Query db err: %s", err.Error())
		os.Exit(1)
	}
	defer rows.Close()
	var face FaceSet
	// facesets := make([]FaceSet, 0)
	for rows.Next() {
		var id int
		// var facesetname interface{}
		err := rows.Scan(&id, &face.FaceSetName, &face.FaceSetID, &face.CreateTime)
		if err != nil {
			db.Close()
			glog.Errorf("scan db err: %s", err.Error())
			os.Exit(1)
		}
		// facesets = append(facesets, face)
		facesetmap[face.FaceSetName] = face.FaceSetID
	}
	if _, ok := facesetmap[config.FaceSetName]; !ok {
		urlStr := config.Aiurl + "/v1/faceSet"
		body := []byte(fmt.Sprintf("{\"faceSetName\":\"%s\"}", config.FaceSetName))
		// resp, err := aicloud.FakeCreateFaceset(urlStr, http.MethodPost, body)
		resp, err := aicloud.CreateFaceset(urlStr, http.MethodPost, body)
		if err != nil {
			glog.Errorf("create faceset err: %s", err.Error())
		}

		// data, err := ioutil.ReadAll(resp.Body)
		// if err != nil {
		// 	db.Close()
		// 	glog.Error(err)
		// 	os.Exit(1)
		// }

		// glog.Infof("create faceset resp: %#v", resp)
		// data := resp
		bdata := make(map[string]interface{})
		err = json.Unmarshal(resp, &bdata)
		if err != nil {
			glog.Errorf("Unmarshal err: %s", err.Error())
		}

		glog.Infof("create faceset data: %#v", bdata)
		stmt, err := db.Prepare("INSERT faceset SET facesetname=?,facesetid=?,createtime=?")
		if err != nil {
			glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
			db.Close()
			glog.Errorf("scan db err: %s", err.Error())
			os.Exit(1)
		}
		defer stmt.Close()
		glog.Infof("config faceset name=%s", config.FaceSetName)
		_, err = stmt.Exec(config.FaceSetName, bdata["faceSetID"].(string), time.Now().UnixNano()/1e6)
		if err != nil {
			glog.Errorf("INSERT faceinfo err: %s", err.Error())
			os.Exit(1)
		}
		facesetmap[config.FaceSetName] = bdata["faceSetID"].(string)
	}

	direc := config.StaticDir + "/" + config.FaceSetName
	if _, err := os.Stat(direc); os.IsNotExist(err) {
		err = os.Mkdir(direc, 0777)
		if err != nil {
			glog.Errorf("mkdir dir err: %s", err.Error())
			os.Exit(1)
		}
	}
	glog.Infof("faceset map: %#v", facesetmap)

	return db
}

func NewManager(config *pkg.Config) *Manager {

	// if _, err := db.Exec("create database if not exists aicloud"); err != nil {
	// 	db.Close()
	// 	glog.Errorf("create db err: %s", err.Error())
	// 	os.Exit(1)
	// }
	aicloud := accessai.NewAccessai()
	facesetmap := make(map[string]string)
	uploadPortal := &model.UploadPortal{
		DetectCache:   list.New(),
		RegisterCache: list.New(),
	}
	m := &Manager{
		CustConfig:   config,
		AiCloud:      aicloud,
		Mydb:         checkFacesetExist(config, aicloud, facesetmap),
		FaceidMap:    facesetmap,
		UploadPortal: uploadPortal,
	}
	glog.Infof("facemap:%#v", m.FaceidMap)
	return m
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

func (m *Manager) detectFace(imagebase64 string) ([]byte, error) {
	// first detect image face
	urlStr := m.CustConfig.Aiurl + "/v1/faceDetect"
	body := []byte(fmt.Sprintf("{\"imageBase64\": \"%s\"}", imagebase64))
	// resp, err := m.AiCloud.FakeFaceSearch(urlStr, http.MethodPost, body)
	resp, err := m.AiCloud.FaceDetect(urlStr, http.MethodPost, body)
	if err != nil {
		glog.Errorf(err.Error())
		return nil, err
	}
	// facedetect := resp
	fdata := make(map[string]interface{})
	err = json.Unmarshal(resp, &fdata)
	if err != nil {
		glog.Errorf(err.Error())
		return nil, err
	}
	if _, ok := fdata["faceNum"]; !ok {
		glog.Errorf("detect face return none")
		return nil, errors.New("detect face return none")
	}
	if fdata["faceNum"].(float64) <= 0.0 {
		return nil, nil
	}
	dfaces := fdata["faces"].([]interface{})
	if len(dfaces) == 0 {
		glog.Errorf("dfaces len is 0")
		return nil, errors.New("dfaces len is 0")
	}

	jdface, err := json.Marshal(dfaces[0])
	if err != nil {
		glog.Errorf("Marshal face err: %s", err)
		return nil, err
	}

	// var dface Face
	// dface.Bound.Height = dfaces[0]["boundingBox"].(map[string]string)["height"]
	// dface.Bound.Width = dfaces[0]["boundingBox"].(map[string]string)["width"]
	// dface.Bound.TopLeftX = dfaces[0]["boundingBox"].(map[string]string)["topLeftX"]
	// dface.Bound.TopLeftY = dfaces[0]["boundingBox"].(map[string]string)["topLeftY"]
	// dface.Confidence = dfaces[0]["confidence"].(string)
	// jdface, err := json.Marshal(dface)
	// if err != nil {
	// 	glog.Errorf(err.Error())
	// 	return nil, err
	// } 
	// glog.Infof("dface: %#v", jdface)
	return jdface, nil
}

func (m *Manager) saveImageToFile(imagename string, imageDecode []byte) error {
	buf := bytes.NewBuffer(nil)
	imageReader := bytes.NewReader(imageDecode)
	if _, err := io.Copy(buf, imageReader); err != nil {
		glog.Errorf("file copy to buf err: %s", err.Error())
	}

	imageaddress := m.CustConfig.StaticDir + "/" + m.CustConfig.FaceSetName + "/" + imagename
	fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		glog.Error(err)
		return err
	}
	defer fileToSave.Close()
	if _, err := io.Copy(fileToSave, buf); err != nil {
		glog.Errorf("buf copy to file err: %s", err.Error())
	}
	return nil
}

func (m *Manager) createFacesetIfNotExist(facesetname string) error {
	// rows, err := db.Query("select * from faceset where facesetname = ?", config.FaceSetName)
	for key := range m.FaceidMap {
		if key == facesetname {
			glog.Infof("faceset %s is already exist!", facesetname)
			return nil
		}
	}

	urlStr := m.CustConfig.Aiurl + "/v1/faceSet"
	body := []byte(fmt.Sprintf("{\"faceSetName\":\"%s\"}", facesetname))
	// resp, err := aicloud.FakeCreateFaceset(urlStr, http.MethodPost, body)
	resp, err := m.AiCloud.CreateFaceset(urlStr, http.MethodPost, body)
	if err != nil {
		glog.Errorf("create faceset err: %s", err.Error())
		return err
	}

	bdata := make(map[string]interface{})
	err = json.Unmarshal(resp, &bdata)
	if err != nil {
		glog.Errorf("Unmarshal err: %s", err.Error())
		return err
	}

	glog.Infof("create faceset data: %#v", bdata)
	stmt, err := m.Mydb.Prepare("INSERT faceset SET facesetname=?,facesetid=?,createtime=?")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
		return err
	}
	defer stmt.Close()
	glog.Infof("config faceset name=%s", facesetname)
	_, err = stmt.Exec(facesetname, bdata["faceSetID"].(string), time.Now().UnixNano()/1e6)
	if err != nil {
		glog.Errorf("INSERT faceinfo err: %s", err.Error())
		return err
	}
	m.FaceidMap[facesetname] = bdata["faceSetID"].(string)

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
	if err := m.createFacesetIfNotExist(facesetname); err != nil {
		return err
	}

	// imageBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	// imageBase64 = ""
	// glog.Infof("image base64: %s", imageBase64)

	// defer upFile.Close()
	uid, err := uuid.NewV4()
	if err != nil {
		glog.Errorf("uuid generate error: %s", err.Error())
	}
	imagename = fmt.Sprintf("%s", uid) + "-" + imagename
	filexist := m.listAlllfiles(m.CustConfig.FaceSetName, imagename)
	if filexist {
		glog.Error("file name already exist")
		return errors.New("file name already exist")
	}

	//构造PicSample对象
	picSample := &model.PicSample{
		Id:          imagename,
		UploadTime:  time.Now().UnixNano() / 1e6,
		Similarity:  make(map[string]int32),
		MostSimilar: "",
	}

	//调用人脸比对API计算相似度
	m.CaculateSimilarity(picSample, imageBase64, facesetname)
	m.CaculateMostSimilarity(picSample)

	if len(picSample.Similarity) > 0 {
		m.UploadPortal.DetectCache.PushBack(picSample)
	} else {
		m.UploadPortal.RegisterCache.PushBack(picSample)
	}

	/** 之前的方法，先留在这做参考，后面可以删除掉这段代码
	// search face in faceset
	// /v1/faceSet/13345/faceSearch?url=http://100.114.203.102/data/2_8.png
	if jdface != nil {
		// save to file
		imageaddress := m.CustConfig.StaticDir + "/" + m.CustConfig.FaceSetName + "/" + imagename
		fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			glog.Error(err)
			return err
		}
		defer fileToSave.Close()
		if _, err := io.Copy(fileToSave, buf); err != nil {
			glog.Errorf("buf copy to file err: %s", err.Error())
		}

		imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + m.CustConfig.FaceSetName + "/" + imagename

		urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[m.CustConfig.FaceSetName] + "/faceSearch?url=" + imageurl
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
		largeface := make(map[string]interface{})
		var largesimilar int64
		if len(faces) > 0 {
			// found := false
			for _, v := range faces {
				face := v.(map[string]interface{})
				similar, err := strconv.ParseInt(face["similarity"].(string), 10, 32)
				if err != nil {
					glog.Errorf("parse error: %s", err.Error())
				}
				if similar >= largesimilar {
					// success
					largesimilar = similar
					largeface = face
				}
				glog.Infof("face similarity: %s", face["similarity"])
			}
			// if !found {
			// 	// todo
			// 	glog.Infof("all similarities are two small")
			// 	// m.insertIntoFacedb(imageaddress, "http://"+m.CustConfig.Host + ":" + m.CustConfig.Port+"/"+handler.Filename, "", "", "")
			// 	// insert unknowfaceinfo
			// 	err = pkg.InsertIntoFacedb(m.Mydb, m.CustConfig.FaceSetName, "", jdface, "", "", "", "", imageaddress, imageurl, time.Now().UnixNano()/1e6, "unknowfaceinfo")
			// 	if err != nil {
			// 		glog.Errorf("INSERT unknowfaceinfo err: %s", err)
			// 		return err
			// 	}
			// }
		}
		if largesimilar >= m.CustConfig.Similarity {
			return m.insertIntoKnow(strconv.FormatInt(largesimilar, 10), imageaddress, imageurl, largeface)
		}

		glog.Infof("no image are match in search")
		// m.insertIntoFacedb(imageaddress, "http://"+m.CustConfig.Host + ":" + m.CustConfig.Port+"/"+handler.Filename, "", "", "")
		err = pkg.InsertIntoFacedb(m.Mydb, m.CustConfig.FaceSetName, "", jdface, "", "", "", "", imageaddress, imageurl, time.Now().UnixNano()/1e6, "", "", "unknowfaceinfo")
		if err != nil {
			glog.Errorf("INSERT unknowfaceinfo err: %s", err)
			return err
		}
	} else {
		glog.Warning("image detect no face")
		return nil
	}
	**/
	return nil
	// glog.Infof("resp :%s", data)
}

func (m *Manager) insertIntoKnow(largesimilar, imageaddress, imageurl string, face map[string]interface{}) error {
	// re := regexp.M
	faceidToS := face["faceID"].(string)
	index := 0
	for k := range faceidToS {
		if string(faceidToS[k]) != "0" {
			index = k
			break
		}
	}
	glog.Infof("index: %d, face:%s", index, faceidToS[index:len(faceidToS)])
	// rows, err := m.Mydb.Query(fmt.Sprintf("select * from facedb where faceid regexp '%s$'", face["faceID"]))
	rows, err := m.Mydb.Query("select * from facedb where faceid = ?", faceidToS[index:len(faceidToS)])
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
		glog.Infof("byte:%#v", faceinteface)
		// jsonface, err := json.Marshal(faceinteface)
		// if err != nil {
		// 	glog.Errorf("Marshal face err: %s", err.Error())
		// 	return err
		// }
		err = pkg.InsertIntoFacedb(m.Mydb, m.CustConfig.FaceSetName, strconv.Itoa(id), faceinteface, "", knowface.Name, knowface.Age, knowface.Address, imageaddress, imageurl, time.Now().UnixNano()/1e6, knowface.ImageURL, largesimilar, "knowfaceinfo")
		if err != nil {
			glog.Errorf("INSERT faceinfo err: %s", err)
			return err
		}
		glog.Infof("found similarity face: %s", face["similarity"])
	} else if len(knowsfaces) == 0 {
		glog.Errorf("facedb has no record of that faceid: %s", face["faceID"])
		return errors.New("facedb has no record of that faceid")
	} else {
		glog.Errorf("facedb has two many faces that match the faceid: %s", face["faceID"])
		return errors.New("facedb has two many faces that match the faceid")
	}
	return nil
}

// func (m *Manager) insertIntoFacedb(facesetname, imageaddress, imageurl, name, age, address string, face interface{}) {
func (m *Manager) insertIntoFacedb(facesetname, imageaddress, imageurl, name, age, address string, face []byte) {
	// /v1/faceSet/13345/addFace
	urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetname] + "/addFace"
	body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\", \"externalImageID\": \"%s\"}", imageurl, m.FaceidMap[facesetname]))
	// resp, err := m.AiCloud.FakeAddFace(urlStr, http.MethodPost, body)
	resp, err := m.AiCloud.AddFace(urlStr, http.MethodPut, body)
	if err != nil {
		glog.Errorf("search face err: %s", err.Error())
	}
	data := resp
	// glog.Infof("string:%s, resp :%#v", data, data)
	if len(resp) == 0 {
		glog.Errorf("add face return 0 length")
		return
	}
	bdata := make(map[string]interface{})
	json.Unmarshal(data, &bdata)

	// glog.Infof("data: %#v", data)
	// glog.Infof("bdata: %#v", bdata["face"])
	// jsonface, err := json.Marshal(bdata["face"])
	// if err != nil {
	// 	glog.Errorf("Marshal face err: %s", err)
	// 	return
	// }

	// jdface, err := json.Marshal(face)
	// glog.Infof("jdface:%#v", jdface)
	// if err != nil {
	// 	glog.Errorf("Marshal face err: %s", err)
	// 	return
	// }

	err = pkg.InsertIntoFacedb(m.Mydb, facesetname, bdata["faceID"].(string), face, "", name, age, address, imageaddress, imageurl, time.Now().UnixNano()/1e6, "", "", "facedb")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
}

// faceset logic
func (m *Manager) faceset(w http.ResponseWriter, r *http.Request) {
	glog.Infof("method: %s", r.Method)
	if r.Method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}

		py := &payload.FacesetRequest{}
		err = json.Unmarshal(data, &py)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("The input body is not valid format.")))
			return
		}

		err = m.createFacesetIfNotExist(py.Faceset.Name);
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("create faceset %s error!", py.Faceset.Name)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("creating faceset %s success or already exist.", py.Faceset.Name)))
		return
	} else {
		glog.Infof("the method %s is not implemented yet.", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("the method %s is not implemented yet.", r.Method)))
		return
	}
}

// upload logic
func (m *Manager) upload(w http.ResponseWriter, r *http.Request) {
	glog.Infof("method: %s", r.Method)
	if r.Method == "GET" {
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, _ := template.ParseFiles("upload.gtpl")
		t.Execute(w, token)
	} else {
		// r.ParseMultipartForm(32 << 20)
		// upFile, handler, err := r.FormFile("uploadfile")
		// if err != nil {
		// 	glog.Error(err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(fmt.Sprintf("500 %s", err)))
		// 	return
		// }

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		bdata := make(map[string]interface{})
		err = json.Unmarshal(data, &bdata)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		filename, ok := bdata["filename"].(string)
		if !ok {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
		}
		imagebody, ok := bdata["imageBase64"].(string)
		if !ok {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
		}
		facesetname, ok := bdata["facesetname"].(string)
		if !ok {
			glog.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("400 %s", err)))
		}

		go m.searchFace(imagebody, filename, facesetname)
		// if  err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(fmt.Sprintf("500 %s", err)))
		// 	return
		// }
		w.WriteHeader(http.StatusOK)
		// fmt.Fprintf(w, "%v", handler.Header)
	}
}

func (m *Manager) getAllfaces(facesetname, table string, start, end, numbers int64, timeby bool) ([]FaceInfo, error) {
	sort := "desc"
	if !timeby {
		sort = "asc"
	}
	qstr := fmt.Sprintf("select * from %s where facesetname='%s' order by createtime %s", table, facesetname, sort)
	glog.Infof(qstr)
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
		knowsfaces = append(knowsfaces, knowface)
		num++
	}
	return knowsfaces, nil
}

func (m *Manager) listfaces(w http.ResponseWriter, r *http.Request) {
	glog.Infof("method: %s", r.Method)
	if r.Method == "GET" {
		urlpara, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusBadRequest)
		}

		glog.Infof("url : %#v", urlpara)
		// data, err := ioutil.ReadAll(r.Body)
		// if err != nil {
		// 	glog.Error(err)
		// 	return
		// }
		// bdata := make(map[string]interface{})
		// // var bdata map[string]interface{}
		// err = json.Unmarshal(data, &bdata)
		// if err != nil {
		// 	glog.Error(err)
		// 	return
		// }
		// start, err := strconv.ParseInt(bdata["start"].(string), 10, 32)
		// end, err := strconv.ParseInt(bdata["end"].(string), 10, 32)
		// number, err := strconv.ParseInt(bdata["numbers"].(string), 10, 32)
		// facesetname := bdata["facesetname"].(string)
		// timeby := bdata["timeby"].(bool)
		// isKnow := bdata["isknown"].(string)

		start, err := strconv.ParseInt(urlpara["start"][0], 10, 32)
		end, err := strconv.ParseInt(urlpara["end"][0], 10, 32)
		number, err := strconv.ParseInt(urlpara["numbers"][0], 10, 32)
		facesetname := urlpara["facesetname"][0]
		timeby, err := strconv.ParseBool(urlpara["timeby"][0])
		isKnow := urlpara["isknown"][0]

		glog.Infof("start:%s, end:%s, number:%s", start, end, number)

		// start, err := r.URL.Query()["start"]
		// start, err := string(r.URL.Query()["start"])
		// end, err := strconv.ParseInt(bdata["end"].(string), 10, 32)
		// number, err := strconv.ParseInt(bdata["numbers"].(string), 10, 32)
		// facesetname := bdata["facesetname"].(string)
		// timeby := bdata["timeby"].(bool)
		// if err != nil {
		// 	glog.Error(err)
		// 	return
		// }
		var table string
		if isKnow == "0" {
			table = "knowfaceinfo"
		} else if isKnow == "1" {
			table = "unknowfaceinfo"
		} else if isKnow == "2" {
			table = "facedb"
		} else {
			glog.Error("unkonw search")
			return
		}
		faces, err := m.getAllfaces(facesetname, table, start, end, number, timeby)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		// glog.Infof("list: %#v", faces)
		// glog.Infof("list: %+v", faces)
		jsonface, err := json.Marshal(faces)
		if err != nil {
			glog.Error(err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", jsonface)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// func (m *Manager) listunknowfaces(w http.ResponseWriter, r *http.Request) {
// 	glog.Infof("method: %s", r.Method)
// 	fmt.Fprintf(w, "%v", "")
// }

func (m *Manager) deletefaces(facesetname string, blist []interface{}, deleteimage bool) error {
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
			urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetname] + "/" + faceid
			_, err := m.AiCloud.DeleteFace(urlStr, http.MethodDelete)
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
			glog.Warning("unkonw face table to delete")
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

func (m *Manager) batchdeletefaces(w http.ResponseWriter, r *http.Request) {
	glog.Infof("method: %s", r.Method)
	if r.Method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		bdata := make(map[string]interface{})
		err = json.Unmarshal(data, &bdata)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		blist := bdata["faces"].([]interface{})
		facesetname := bdata["facesetname"].(string)
		err = m.deletefaces(facesetname, blist, true)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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
		m.deletefaces(facesetname, blist, false)
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
		} else {
			glog.Warning("unknow table to update")
		}
	}
	return nil
}

func (m *Manager) faceinforegister(w http.ResponseWriter, r *http.Request) {
	glog.Infof("method: %s", r.Method)
	if r.Method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		bdata := make(map[string]interface{})
		err = json.Unmarshal(data, &bdata)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		blist := bdata["faceinfo"].(map[string]interface{})
		err = m.updateface(blist)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
		}
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	// w.Write([]byte(fmt.Sprintf("500 %s", )))
}

// upload logic
func (m *Manager) addFace(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method) //获取请求的方法
	if r.Method == "GET" {
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, _ := template.ParseFiles("upload.gtpl")
		t.Execute(w, token)
	} else {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}
		bdata := make(map[string]interface{})
		err = json.Unmarshal(data, &bdata)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", err)))
			return
		}

		// r.ParseMultipartForm(32 << 20)
		// file, handler, err := r.FormFile("uploadfile")
		// if err != nil {
		// 	glog.Error(err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(fmt.Sprintf("500 uploadfile not found")))
		// 	return
		// }
		// defer file.Close()

		// buf := bytes.NewBuffer(nil)
		// if _, err := io.Copy(buf, file); err != nil {
		// 	glog.Error(err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	w.Write([]byte(fmt.Sprintf("500 io copy failed")))
		// 	return
		// }

		// defer upFile.Close()
		uid, err := uuid.NewV4()
		if err != nil {
			glog.Errorf("uuid generate error: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 generate uuid error")))
			return
		}
		imagename := fmt.Sprintf("%s", uid) + "-" + bdata["imagename"].(string)
		filexist := m.listAlllfiles(bdata["facesetname"].(string), imagename)
		if filexist {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 %s", "file name already exist")))
			return
		}

		// first detect image face
		imagedecode, err := base64.StdEncoding.DecodeString(bdata["imagebase64"].(string))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 decode image err")))
			return
		}
		jdface, err := m.detectFace(bdata["imagebase64"].(string))
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 cannot detect face")))
			return
		}
		glog.Infof("dface: %#v", jdface)

		// save to file
		if jdface != nil {
			imageaddress := m.CustConfig.StaticDir + "/" + bdata["facesetname"].(string) + "/" + imagename
			fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
			if err != nil {
				glog.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("500 can not create file")))
				return
			}
			defer fileToSave.Close()
			if _, err := fileToSave.Write(imagedecode); err != nil {
				glog.Errorf("buf copy to file err: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("500 can not create file")))
				return
			}

			imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + bdata["facesetname"].(string) + "/" + imagename
			m.insertIntoFacedb(bdata["facesetname"].(string), imageaddress, imageurl, bdata["name"].(string), bdata["age"].(string), bdata["address"].(string), jdface)
			// body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\"}", imageurl))
			// resp, err := m.AiCloud.FakeFaceSearch(urlStr, http.MethodPost, body)
			// resp, err := m.AiCloud.FaceSearch(urlStr, http.MethodGet, nil)
			// if err != nil {
			// 	glog.Errorf(err.Error())
			// 	return err
			// }
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 the face")))
			return
		}
		w.WriteHeader(http.StatusOK)

		// fmt.Fprintf(w, "%v", handler.Header)
		// f, err := os.OpenFile("./test/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)  // 此处假设当前目录下已存在test目录
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
		// defer f.Close()
		// io.Copy(f, file)
	}
}

func (m *Manager) timeToRemoveImages() {

}

func main() {
	config := pkg.InitConfig()
	m := NewManager(config)

	//定时调度
	cron := cron.New()
	spec := "*/20 * * * * ?"
	cron.AddFunc(spec, func() {
		m.cacheScheduler()
	})
	cron.Start()
	//

	fs := http.FileServer(http.Dir(config.StaticDir))
	http.Handle("/", fs)
	// go glog.Error(http.ListenAndServe(":9091", http.FileServer(http.Dir(config.StaticDir))))
	// http.HandleFunc(config.StaticDir+"/", func(w http.ResponseWriter, r *http.Request) {
	// 	glog.Infof("r.URL.Path: %s", r.URL.Path[1:])
	//     http.ServeFile(w, r, r.URL.Path[1:])
	// })
	http.HandleFunc("/api/v1/faceset", m.faceset)
	http.HandleFunc("/api/v1/faces/upload", m.upload)
	http.HandleFunc("/api/v1/faces/add", m.addFace)
	http.HandleFunc("/api/v1/faces", m.listfaces)
	// http.HandleFunc("/api/v1/listunknowfaces", m.listunknowfaces)
	http.HandleFunc("/api/v1/faces/delete", m.batchdeletefaces)
	http.HandleFunc("/api/v1/faces/register", m.faceinforegister)

	glog.Infof("Serving %s on HTTP port: %s\n", config.StaticDir, config.Port)
	go m.timeToRemoveImages()
	glog.Error(http.ListenAndServe(":"+config.Port, nil))
}

//计算相似度
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
	jdface, err := m.detectFace(imageBase64)
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Infof("dface: %#v", jdface)

	// search face in faceset
	// /v1/faceSet/13345/faceSearch?url=http://100.114.203.102/data/2_8.png
	//如果检测到人脸了，就先存储到文件系统，然后进行1：N的人脸搜索
	if jdface != nil {

		//这里的sample.Id 就是图片在文件系统中存储的名字，格式为 UUID + 上传的原始图片名.
		imageaddress := m.CustConfig.StaticDir + "/" + m.CustConfig.FaceSetName + "/" + picSample.Id
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
		imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + m.CustConfig.FaceSetName + "/" + picSample.Id
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
	curMostSimilarityKey	:= ""
	if len(sample.Similarity) > 0{
		for k,v := range sample.Similarity {
			if v > curMostSimilarityValue {
				curMostSimilarityKey = k
				curMostSimilarityValue = v
			}
		}
		sample.MostSimilar = curMostSimilarityKey
	}
}

//计算当前图片与缓存集的相似度，注意该图片也在缓存集中，计算时需要过滤掉自己与自己的相似度
func (m *Manager) caculateSimilarityWithCache(picSample *model.PicSample, cacheList *list.List, imageBase64 string, facesetName string) model.SRList {
	m.CaculateSimilarity(picSample, imageBase64, facesetName)
	result := model.SRList{}

	for k, v := range picSample.Similarity {
		//如果相似度大于99，认为是缓存中存在该图片，需要过滤处理
		if v < 99 {
			to := m.getPicSample(k, cacheList)
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

//按id从list中找到该对象
func (m *Manager) getPicSample(id string, list *list.List) *model.PicSample {
	if list.Len() > 0 {
		for e:= list.Front(); e!=nil; e=e.Next() {
			if e.Value.(model.PicSample).Id == id {
				return e.Value.(*model.PicSample)
			}
		}
	}
	return nil
}

//保存到facedb数据库
func (m *Manager) saveToRegisterDB(picSample *model.PicSample, facesetName string) error {
	err := pkg.InsertIntoFacedb(m.Mydb, facesetName, picSample.Id, nil, picSample.ImageBase64, "","","","","",time.Now().UnixNano()/1e6, "", "","facedb")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
	return err
}

//保存到已识别的knowfaceinfo数据库
func (m *Manager) saveToDetectDB(picSample *model.PicSample, facesetName string) error {
	err := pkg.InsertIntoFacedb(m.Mydb, facesetName, picSample.Id, nil, picSample.ImageBase64, "","","","","",time.Now().UnixNano()/1e6, "","","knowfaceinfo")
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	}
	return err
}

//从相似矩阵获取最相似列表
func (m *Manager) caculateMostSimilarity(matrix *model.SRMatrix) model.SRList {
	result := model.SRList{}

	srFromMap := model.SRFromMap{}
	srToMap := model.SRToMap{}
	srMap := model.SRMap{}

	for _,srList := range *matrix {
		for _,sr := range srList {
			if v, ok := srFromMap[sr.From]; ok {
				if sr.Similary > v {
					srFromMap[sr.From] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				}
			} else {
				srFromMap[sr.From] = sr.Similary
				tempToMap := model.SRToMap{}
				tempToMap[sr.To] = sr.Similary
				srMap[sr.From] = tempToMap
			}

			if v, ok := srToMap[sr.To]; ok {
				if sr.Similary > v {
					srToMap[sr.To] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				} else {
					srToMap[sr.To] = sr.Similary
					tempToMap := model.SRToMap{}
					tempToMap[sr.To] = sr.Similary
					srMap[sr.From] = tempToMap
				}
			}
		}
	}

	for from, toMap := range srMap {
		for to, similarity := range toMap {
			var sr model.SimilaryRelation
			sr.From = from
			sr.To = to
			sr.Similary = similarity
			result = append(result, sr)
		}
	}
	return result
}

//将缓存的图片集放到云上
func (m *Manager) addCacheFaceSet(cacheList *list.List, facesetName string) error {
	if cacheList.Len() > 0 {
		for pic := cacheList.Front(); pic != nil;pic = pic.Next() {
			// first detect image face
			imagedecode, err := base64.StdEncoding.DecodeString(pic.Value.(model.PicSample).ImageBase64)
			if err != nil {
				glog.Error(err)
				return err
			}
			jdface, err := m.detectFace(pic.Value.(model.PicSample).ImageBase64)
			if err != nil {
				glog.Error(err)
				return err
			}
			glog.Infof("dface: %#v", jdface)

			// save to file
			if jdface != nil {
				imageaddress := m.CustConfig.StaticDir + "/" + facesetName + "/" + pic.Value.(model.PicSample).Id
				fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
				if err != nil {
					glog.Error(err)
					return err
				}
				defer fileToSave.Close()
				if _, err := fileToSave.Write(imagedecode); err != nil {
					glog.Errorf("buf copy to file err: %s", err.Error())
					return err
				}

				imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + facesetName + "/" + pic.Value.(model.PicSample).Id
				// /v1/faceSet/13345/addFace
				urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetName] + "/addFace"
				body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\", \"externalImageID\": \"%s\"}", imageurl, m.FaceidMap[facesetName]))
				// resp, err := m.AiCloud.FakeAddFace(urlStr, http.MethodPost, body)
				resp, err := m.AiCloud.AddFace(urlStr, http.MethodPut, body)
				if err != nil {
					glog.Errorf("search face err: %s", err.Error())
				}
				data := resp
				// glog.Infof("string:%s, resp :%#v", data, data)
				if len(resp) == 0 {
					glog.Errorf("add face return 0 length")
					return err
				}
				bdata := make(map[string]interface{})
				json.Unmarshal(data, &bdata)


			} else {
				return err
			}
		}
	}
	return nil
}

//从云上删除缓存的图像集
func (m *Manager) deleteCacheFaceSet(cacheList *list.List, facesetName string) error {
	if cacheList.Len() > 0 {
		for pic := cacheList.Front(); pic != nil; pic = pic .Next() {
			// delete from faceset
			urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetName] + "/" + pic.Value.(model.PicSample).Id
			_, err := m.AiCloud.DeleteFace(urlStr, http.MethodDelete)
			if err != nil {
				glog.Error(err)
				return err
			}

			//delete from os
			imageurl := "http://" + m.CustConfig.PublicHost + ":" + m.CustConfig.Port + "/" + facesetName + "/" + pic.Value.(model.PicSample).Id
			glog.Infof("image location:%s", strings.Split(imageurl, ":"+m.CustConfig.Port)[1])
			imageaddress := m.CustConfig.StaticDir + strings.Split(imageurl, ":"+m.CustConfig.Port)[1]
			e := os.Remove(imageaddress)
			if e != nil {
				glog.Error("remove err:%s", e)
			}
		}
	}
	return nil
}

//缓存调度器
func (m *Manager) cacheScheduler() {
	//分两个任务调度
	var lock sync.RWMutex

	//1.注册缓存
	lock.Lock()
	tempRegisterCache := m.UploadPortal.TempRegisterCache
	tempRegisterCache.PushBack(m.UploadPortal.RegisterCache)
	m.UploadPortal.RegisterCache.Init()
	lock.Unlock()

	if tempRegisterCache.Len() > 0 {
		m.addCacheFaceSet(tempRegisterCache, "cacheFaceset")
		for {
			if tempRegisterCache.Len() == 0 {
				break
			}

			picSample := tempRegisterCache.Back()
			similaryRelations := m.caculateSimilarityWithCache(picSample.Value.(*model.PicSample), tempRegisterCache, picSample.Value.(*model.PicSample).ImageBase64,"cacheFaceset")

			for i:= 0; i<len(similaryRelations); i=i+1 {
				var n *list.Element
				for e:=tempRegisterCache.Front(); e!= nil; e=n {
					if e.Value.(model.PicSample).Id == similaryRelations[i].To.Id {
						n = e.Next()
						tempRegisterCache.Remove(e)
					}
				}
			}
			tempRegisterCache.Remove(picSample)
		}
		m.deleteCacheFaceSet(tempRegisterCache, "cacheFaceset")
	}

	//2.识别缓存
	lock.Lock()
	tempDetectCache := m.UploadPortal.TempDetectCache
	tempDetectCache.PushBackList(m.UploadPortal.DetectCache)
	m.UploadPortal.DetectCache.Init()
	lock.Unlock()

	if tempDetectCache.Len() > 0 {
		m.addCacheFaceSet(tempDetectCache, "cacheFaceset")

		srMetrix := model.SRMatrix{}
		lastSaveMap := make(model.LastSaveMap)

		for e:=tempDetectCache.Front(); e!=nil; e=e.Next() {
			similaryRelations := m.caculateSimilarityWithCache(e.Value.(*model.PicSample), tempDetectCache, e.Value.(*model.PicSample).ImageBase64, "cacheFaceset")
			srMetrix = append(srMetrix, similaryRelations)
		}

		if len(srMetrix) > 0 {
			mostSimiliarityRelations := m.caculateMostSimilarity(&srMetrix)

			tempDetectCache.Init()

			for i:=0; i< len(mostSimiliarityRelations); i = i+1 {
				fromPic := mostSimiliarityRelations[i].From
				toPic := mostSimiliarityRelations[i].To

				if fromPic.UploadTime - lastSaveMap[toPic] > 30*1000 {
					m.saveToDetectDB(fromPic, m.CustConfig.FaceSetName)

					lastSaveMap[toPic] = fromPic.UploadTime
				} else {
					tempDetectCache.PushBack(fromPic)
				}
			}
		}
		m.deleteCacheFaceSet(tempDetectCache, "cacheFaceset")
	}
}
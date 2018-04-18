package manager

import (
	"edge-for-image/pkg/db"
	"edge-for-image/pkg/model"
	"edge-for-image/pkg/payload"
	"encoding/json"
	"errors"
	"io/ioutil"
	// "path/filepath"
	// "flag"
	"fmt"

	"database/sql"
	"net/http"
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
	CustConfig    *pkg.Config
	AiCloud       *accessai.Accessai
	Mydb          *sql.DB
	FaceidMap     map[string]string
	UploadPortal  map[string]model.Caches //to delete
	DetectCache   map[string][]model.PicSample
	RegistCache   map[string][]model.PicSample
	LastSaveMap   map[string]int64
	CloseToRegist map[string]bool
	RingBuffer    model.Queen
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

//func (m *Manager) saveImageToFile(imagename string, imageDecode []byte) error {
//	buf := bytes.NewBuffer(nil)
//	imageReader := bytes.NewReader(imageDecode)
//	if _, err := io.Copy(buf, imageReader); err != nil {
//		glog.Errorf("file copy to buf err: %s", err.Error())
//	}
//
//	imageaddress := m.CustConfig.StaticDir + "/" + m.CustConfig.FaceSetName + "/" + imagename
//	fileToSave, err := os.OpenFile(imageaddress, os.O_WRONLY|os.O_CREATE, 0777)
//	if err != nil {
//		glog.Error(err)
//		return err
//	}
//	defer fileToSave.Close()
//	if _, err := io.Copy(fileToSave, buf); err != nil {
//		glog.Errorf("buf copy to file err: %s", err.Error())
//	}
//	return nil
//}

func (m *Manager) CreateFacesetIfNotExist(facesetname string) error {
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

	//if exist from ai, then it exist.
	if strings.Contains(string(resp), "exist") {
		glog.Infof("the faceset already exist")
		return nil
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
	if err := m.CreateFacesetIfNotExist(facesetname); err != nil {
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
	glog.Infof("the imagename is %s", imagename)
	//filexist := m.listAlllfiles(facesetname, imagename)
	//if filexist {
	//	glog.Error("file name already exist")
	//	return errors.New("file name already exist")
	//}

	//构造PicSample对象
	picSample := &model.PicSample{
		Id:          imagename,
		UploadTime:  time.Now().UnixNano() / 1e6,
		Similarity:  make(map[string]int32),
		MostSimilar: 0,
		ImageBase64: imageBase64,
	}

	//调用人脸比对API计算相似度
	m.CaculateSimilarity(picSample, imageBase64, facesetname)
	m.CaculateMostSimilarity(picSample)

	if picSample.MostSimilar > int32(m.CustConfig.Similarity) {
		m.DetectCache[facesetname] = append(m.DetectCache[facesetname], *picSample)
	} else {
		m.RegistCache[facesetname] = append(m.RegistCache[facesetname], *picSample)
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

func (m *Manager) insertIntoKnow(largesimilar, imageaddress, imageurl string, faceid string, facesetname string) error {
	// re := regexp.M
	//faceidToS := face["faceID"].(string)
	//index := 0
	//for k := range faceidToS {
	//	if string(faceidToS[k]) != "0" {
	//		index = k
	//		break
	//	}
	//}
	//glog.Infof("index: %d, face:%s", index, faceidToS[index:len(faceidToS)])
	// rows, err := m.Mydb.Query(fmt.Sprintf("select * from facedb where faceid regexp '%s$'", face["faceID"]))
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
		glog.Infof("byte:%#v", faceinteface)
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

	err = db.InsertIntoFacedb(m.Mydb, facesetname, bdata["faceID"].(string), face, "", name, age, address, imageaddress, imageurl, time.Now().UnixNano()/1e6, "", "", "facedb")
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

func (m *Manager) timeToRemoveImages() {

}
func (m *Manager) DeleteFaceset(facesetname string) {
	urlStr := m.CustConfig.Aiurl + "/v1/faceSet"
	body := []byte(fmt.Sprintf("{\"faceSetName\":\"%s\"}", facesetname))
	m.AiCloud.DeleteFaceset(urlStr, http.MethodDelete, body)
}

func (m *Manager) FaceVerify(url1 string, url2 string) (resp payload.FaceVerifyResponse) {
	urlStr := m.CustConfig.Aiurl + "/v1/faceVerify"
	req := &payload.FaceVerifyRequest{
		Image1URL: url1,
		Image2Url: url2,
	}
	//todo: here igored error.
	body, _ := json.Marshal(req)
	data, _ := m.AiCloud.FaceVerify(urlStr, http.MethodPost, body)
	json.Unmarshal(data, &resp)
	return resp
}

func (m *Manager) AddFaceToSet(imageUrl string, facesetname string) (resp payload.AddFaceResponse) {
	urlStr := m.CustConfig.Aiurl + "/v1/faceSet/" + m.FaceidMap[facesetname] + "/addFace"
	body := []byte(fmt.Sprintf("{\"imageUrl\": \"%s\", \"externalImageID\": \"%s\"}", imageUrl, m.FaceidMap[facesetname]))
	data, _ := m.AiCloud.AddFace(urlStr, http.MethodPut, body)
	json.Unmarshal(data, &resp)
	return resp
}

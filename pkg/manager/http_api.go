package manager

import (
	"edge-for-image/pkg/payload"
	"encoding/json"
	"github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"net/url"
	// "path/filepath"
	// "flag"
	"fmt"
	// "log"
	"crypto/md5"
	"html/template"
	"net/http"
	"strconv"

	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
)

// faceset logic
func (m *Manager) Faceset(w http.ResponseWriter, r *http.Request) {
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

		glog.Infof("request is %v", py)
		err = m.CreateFacesetIfNotExist(py.Name);
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("create faceset %s error!", py.Name)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("{\"status\": \"success\"}")))
		return
	} else {
		glog.Infof("the method %s is not implemented yet.", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("the method %s is not implemented yet.", r.Method)))
		return
	}
}

// upload logic
func (m *Manager) Upload(w http.ResponseWriter, r *http.Request) {
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
		glog.Infof("====================================================== start here ====================")

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

		glog.Infof("image upload time: %s", getTimeStamp(filename))

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
func getTimeStamp(filename string) (tm string) {
	if len(filename) != 17 {
		return ""
	}

	i, err := strconv.ParseInt(filename[0:12], 10, 64)
	if err != nil {
		return ""
	}
	tm1 := time.Unix(i/1000, (i%1000)*1000)
	return tm1.Format("2006-01-02 15:04:05.000")
}

func (m *Manager) Batchdeletefaces(w http.ResponseWriter, r *http.Request) {
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
		err = m.Deletefaces(facesetname, blist, true)
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

// upload logic
func (m *Manager) AddFace(w http.ResponseWriter, r *http.Request) {
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

		uid, err := uuid.NewV4()
		if err != nil {
			glog.Errorf("uuid generate error: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 generate uuid error")))
			return
		}
		imagename := fmt.Sprintf("%s", uid) + "-" + bdata["imagename"].(string)
		glog.Infof("regist imagename is %s", imagename)

		inputKey := bdata["facesetname"].(string) + "/" + imagename
		err = m.UploadImageToObs(inputKey, bdata["imagebase64"].(string))
		if err != nil {
			glog.Errorf("Upload Image %s to OBS failed!", inputKey)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 upload obs err")))
			return
		}

		m.insertIntoFacedb(bdata["facesetname"].(string), inputKey, inputKey, bdata["name"].(string), bdata["age"].(string), bdata["address"].(string), nil)
		w.WriteHeader(http.StatusOK)

	}
}

func (m *Manager) Listfaces(w http.ResponseWriter, r *http.Request) {
	//glog.Infof("method: %s", r.Method)
	if r.Method == "GET" {
		urlpara, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusBadRequest)
		}

		//glog.Infof("url : %#v", urlpara)
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

		//glog.Infof("start:%s, end:%s, number:%s", start, end, number)

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
		faces, err := m.GetAllfaces(facesetname, table, start, end, number, timeby)
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

func (m *Manager) Faceinforegister(w http.ResponseWriter, r *http.Request) {
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

// func (m *Manager) listunknowfaces(w http.ResponseWriter, r *http.Request) {
// 	glog.Infof("method: %s", r.Method)
// 	fmt.Fprintf(w, "%v", "")
// }

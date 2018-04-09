package manager

import (
	"edge-for-image/pkg/payload"
	"encoding/base64"
	"encoding/json"
	"github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"net/url"
	"os"

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

		err = m.CreateFacesetIfNotExist(py.Faceset.Name);
		if err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("create faceset %s error!", py.Faceset.Name)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("{\"status\": \"success\"")))
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
		//filexist := m.listAlllfiles(bdata["facesetname"].(string), imagename)
		//if filexist {
		//	w.WriteHeader(http.StatusInternalServerError)
		//	w.Write([]byte(fmt.Sprintf("500 %s", "file name already exist")))
		//	return
		//}

		// first detect image face
		imagedecode, err := base64.StdEncoding.DecodeString(bdata["imagebase64"].(string))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 decode image err")))
			return
		}
		jdface, err := m.DetectFace(bdata["imagebase64"].(string))
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

func (m *Manager) Listfaces(w http.ResponseWriter, r *http.Request) {
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

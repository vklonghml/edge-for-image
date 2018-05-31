package main

import (
	"database/sql"
	"edge-for-image/pkg"
	"edge-for-image/pkg/accessai"
	"edge-for-image/pkg/manager"
	"edge-for-image/pkg/model"
	"edge-for-image/pkg/scheduler"
	"github.com/golang/glog"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"time"
)

func refreshToken(config *pkg.Config, m *manager.Manager) {
	for i := 0; i < 5; i++ {
		t, p, err := m.IAMClient.GetToken()
		if err == nil {
			glog.Infof("success get token: %s, project: %s", t, p)
			break
		}
		glog.Errorf("error to refresh token with err: %s", err)
		time.Sleep(5 * time.Second)
	}
	for {
		time.Sleep(time.Duration(config.IAMRefresh) * time.Second)
		for {
			t, p, err := m.IAMClient.GetToken()
			if err == nil {
				glog.Infof("success get token: %s, project: %s", t, p)
				break
			}
			glog.Errorf("error to refresh token with err: %s", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func main() {
	config := pkg.InitConfig()
	m := NewManager(config)
	s := scheduler.Scheduler{}

	//定时调度
	//cron := cron.New()
	//registSpec := fmt.Sprintf("*/%d * * * * ?", m.CustConfig.RegistPeriodSec)
	//detectSpec := fmt.Sprintf("*/%d * * * * ?", m.CustConfig.DetectPeriodSec)
	//cron.AddFunc(registSpec, func() {
	//	s.CacheSchedulerRegist(m)
	//})
	//cron.AddFunc(detectSpec, func() {
	//	s.CacheSchedulerDetect(m)
	//})
	//cron.Start()
	//
	go refreshToken(config, m)
	go newLoop(m, s)

	fs := http.FileServer(http.Dir(config.StaticDir))
	http.Handle("/", fs)
	// go glog.Error(http.ListenAndServe(":9091", http.FileServer(http.Dir(config.StaticDir))))
	// http.HandleFunc(config.StaticDir+"/", func(w http.ResponseWriter, r *http.Request) {
	// 	glog.Infof("r.URL.Path: %s", r.URL.Path[1:])
	//     http.ServeFile(w, r, r.URL.Path[1:])
	// })
	http.HandleFunc("/api/v1/faceset", m.Faceset)
	http.HandleFunc("/api/v1/faces/upload", m.Upload)
	http.HandleFunc("/api/v1/faces/add", m.AddFace)
	http.HandleFunc("/api/v1/faces", m.Listfaces)
	// http.HandleFunc("/api/v1/listunknowfaces", m.listunknowfaces)
	http.HandleFunc("/api/v1/faces/delete", m.Batchdeletefaces)
	http.HandleFunc("/api/v1/faces/register", m.Faceinforegister)

	glog.Infof("Serving %s on HTTP port: %s\n", config.StaticDir, config.Port)
	//go m.timeToRemoveImages()
	glog.Error(http.ListenAndServe(":"+config.Port, nil))

}

func NewManager(config *pkg.Config) *manager.Manager {

	// if _, err := db.Exec("create database if not exists aicloud"); err != nil {
	// 	db.Close()
	// 	glog.Errorf("create db err: %s", err.Error())
	// 	os.Exit(1)
	// }
	aicloud := accessai.NewAccessai()
	facesetmap := make(map[string]string)
	m := &manager.Manager{
		CustConfig:    config,
		AiCloud:       aicloud,
		IAMClient:     accessai.NewIAMClient(config.IAMURL, config.IAMName, config.IAMPassword, config.IAMProject, config.IAMDomain),
		Mydb:          checkFacesetExist(config, aicloud, facesetmap),
		FaceidMap:     facesetmap,
		RegistCache:   make(map[string][]model.PicSample),
		DetectCache:   make(map[string][]model.PicSample),
		RegistThread:  make(map[string]int32),
		DetectThread:  make(map[string]int32),
		LastSaveMap:   make(map[string]int64),
		CloseToRegist: make(map[string]bool),
		RingBuffer:    make(map[string]*model.Queen),
	}
	glog.Infof("facemap:%#v", m.FaceidMap)
	return m
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
	var face manager.FaceSet
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
	//if _, ok := facesetmap[config.FaceSetName]; !ok {
	//	urlStr := config.Aiurl + "/v1/faceSet"
	//	body := []byte(fmt.Sprintf("{\"faceSetName\":\"%s\"}", config.FaceSetName))
	//	// resp, err := aicloud.FakeCreateFaceset(urlStr, http.MethodPost, body)
	//	resp, err := aicloud.CreateFaceset(urlStr, http.MethodPost, body)
	//	if err != nil {
	//		glog.Errorf("create faceset err: %s", err.Error())
	//	}
	//
	//	// data, err := ioutil.ReadAll(resp.Body)
	//	// if err != nil {
	//	// 	db.Close()
	//	// 	glog.Error(err)
	//	// 	os.Exit(1)
	//	// }
	//
	//	// glog.Infof("create faceset resp: %#v", resp)
	//	// data := resp
	//	bdata := make(map[string]interface{})
	//	err = json.Unmarshal(resp, &bdata)
	//	if err != nil {
	//		glog.Errorf("Unmarshal err: %s", err.Error())
	//	}
	//
	//	glog.Infof("create faceset data: %#v", bdata)
	//	stmt, err := db.Prepare("INSERT faceset SET facesetname=?,facesetid=?,createtime=?")
	//	if err != nil {
	//		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
	//		db.Close()
	//		glog.Errorf("scan db err: %s", err.Error())
	//		os.Exit(1)
	//	}
	//	defer stmt.Close()
	//	glog.Infof("config faceset name=%s", config.FaceSetName)
	//	_, err = stmt.Exec(config.FaceSetName, bdata["faceSetID"].(string), time.Now().UnixNano()/1e6)
	//	if err != nil {
	//		glog.Errorf("INSERT faceinfo err: %s", err.Error())
	//		os.Exit(1)
	//	}
	//	facesetmap[config.FaceSetName] = bdata["faceSetID"].(string)
	//}
	//
	//direc := config.StaticDir + "/" + config.FaceSetName
	//if _, err := os.Stat(direc); os.IsNotExist(err) {
	//	err = os.Mkdir(direc, 0777)
	//	if err != nil {
	//		glog.Errorf("mkdir dir err: %s", err.Error())
	//		os.Exit(1)
	//	}
	//}
	glog.Infof("faceset map: %#v", facesetmap)

	return db
}

func newLoop(m *manager.Manager, s scheduler.Scheduler) {
	for {
		//glog.Infof("starting go routine")
		for k, v := range m.RegistThread {
			if v == 1 {
				go s.LoopHandleRegistCache(m, k)
				m.RegistThread[k] = 2
				m.RingBuffer[k] = model.MakeQueen(5)
				glog.Infof("start a new regist thread for faceset: %s.", k)
			}
		}
		for k, v := range m.DetectThread {
			if v == 1 {
				go s.LoopHandleDetectCache(m, k)
				m.DetectThread[k] = 2
				glog.Infof("start a new detect thread for faceset: %s.", k)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

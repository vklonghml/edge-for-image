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
	_ "net/http/pprof"
	_ "github.com/rakyll/gom/http"
	"time"
	"edge-for-image/pkg/obs"
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

var obsClient *obs.ObsClient

func getObsClient() *obs.ObsClient {
	var err error
	if obsClient == nil {
		obsClient, err = obs.New(pkg.Config0.AK, pkg.Config0.SK, pkg.Config0.OBSEndpoint)
		if err != nil {
			panic(err)
		}
	}
	return obsClient
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
		ObsClient:     getObsClient(),
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

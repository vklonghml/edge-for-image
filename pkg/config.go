package pkg

import (
	"flag"
)

type Config struct {
	Host       string
	PublicHost string
	Port       string
	StaticDir  string
	Aiurl      string
	//FaceSetName      string
	DBConnStr       string
	Diskthreshold   int
	Similarity      int
	PicWaitSec      int64
	RegistPeriodSec int
	DetectPeriodSec int
	RegistCacheSize int
	DetectCacheSize int
	AutoRegistSize int
}

func InitConfig() *Config {
	config := Config{}
	flag.StringVar(&config.Host, "host", "0.0.0.0", "host to serve on")
	flag.StringVar(&config.PublicHost, "public-host", "127.0.0.1", "floating ip to serve on")
	flag.StringVar(&config.Port, "port", "9090", "port to serve on")
	flag.StringVar(&config.StaticDir, "static-dir", ".", "the directory of static file to host")
	flag.StringVar(&config.Aiurl, "aiurl", "", "url of ai service")
	//flag.StringVar(&config.FaceSetName, "faceset-name", "", "faceset to work with")
	flag.StringVar(&config.DBConnStr, "db-conn-str", "", "db connection string (user:password@tcp(ip:port)/dbname)")
	flag.IntVar(&config.Diskthreshold, "disk-threshod", 80, "images data store in disk to trigger to rm images")
	flag.IntVar(&config.Similarity, "similarity", 92, "similarity to judge two image are similar")

	flag.Int64Var(&config.PicWaitSec, "pic-wait-sec", 30, "the second for a pic to be detected or registered")
	flag.IntVar(&config.RegistPeriodSec, "regist-period-sec", 20, "register cache scheduler period")
	flag.IntVar(&config.DetectPeriodSec, "detect-period-sec", 20, "detector cache scheduler period")

	flag.IntVar(&config.RegistCacheSize, "regist-cache-size", 20, "max regist cache size")
	flag.IntVar(&config.DetectCacheSize, "detect-cache-size", 20, "max detect cache size")
	flag.IntVar(&config.AutoRegistSize, "auto-regist-size", 100, "auto regist size")

	flag.Parse()
	return &config
}

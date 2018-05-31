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
	AutoRegistSize  int
	// iam
	IAMURL      string
	IAMName     string
	IAMPassword string
	IAMProject  string
	IAMDomain   string
	IAMRefresh  int
	// obs
	OBSEndpoint   string
	OBSBucketName string
	AK            string
	SK            string
}

var Config0 = Config{}

func InitConfig() *Config {
	flag.StringVar(&Config0.Host, "host", "0.0.0.0", "host to serve on")
	flag.StringVar(&Config0.PublicHost, "public-host", "127.0.0.1", "floating ip to serve on")
	flag.StringVar(&Config0.Port, "port", "9090", "port to serve on")
	flag.StringVar(&Config0.StaticDir, "static-dir", ".", "the directory of static file to host")
	flag.StringVar(&Config0.Aiurl, "aiurl", "", "url of ai service")
	//flag.StringVar(&Config0.FaceSetName, "faceset-name", "", "faceset to work with")
	flag.StringVar(&Config0.DBConnStr, "db-conn-str", "", "db connection string (user:password@tcp(ip:port)/dbname)")
	flag.IntVar(&Config0.Diskthreshold, "disk-threshod", 80, "images data store in disk to trigger to rm images")
	flag.IntVar(&Config0.Similarity, "similarity", 92, "similarity to judge two image are similar")

	flag.Int64Var(&Config0.PicWaitSec, "pic-wait-sec", 30, "the second for a pic to be detected or registered")
	flag.IntVar(&Config0.RegistPeriodSec, "regist-period-sec", 20, "register cache scheduler period")
	flag.IntVar(&Config0.DetectPeriodSec, "detect-period-sec", 20, "detector cache scheduler period")

	flag.IntVar(&Config0.RegistCacheSize, "regist-cache-size", 20, "max regist cache size")
	flag.IntVar(&Config0.DetectCacheSize, "detect-cache-size", 20, "max detect cache size")
	flag.IntVar(&Config0.AutoRegistSize, "auto-regist-size", 100, "auto regist size")

	flag.StringVar(&Config0.IAMURL, "iam-url", ".", "iam url")
	flag.StringVar(&Config0.IAMName, "iam-name", ".", "iam name")
	flag.StringVar(&Config0.IAMPassword, "iam-password", ".", "iam password")
	flag.StringVar(&Config0.IAMProject, "iam-project", ".", "iam project")
	flag.StringVar(&Config0.IAMDomain, "iam-domain", ".", "iam domain")
	flag.IntVar(&Config0.IAMRefresh, "iam-refresh", 3600, "iam refresh interval")

	flag.StringVar(&Config0.OBSEndpoint, "obs-endpoint", ".", "obs endpoint")
	flag.StringVar(&Config0.OBSBucketName, "obs-bucket-name", ".", "obs bucket name")
	flag.StringVar(&Config0.AK, "ak", ".", "ak")
	flag.StringVar(&Config0.SK, "sk", ".", "sk")

	flag.Parse()
	return &Config0
}

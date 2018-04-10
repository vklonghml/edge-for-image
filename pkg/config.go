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
	DBConnStr     string
	Diskthreshold int
	Similarity    int
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

	flag.Parse()
	return &config
}

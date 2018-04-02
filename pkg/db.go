package pkg

import (
	"database/sql"
	"fmt"

	"github.com/golang/glog"
	_ "github.com/go-sql-driver/mysql"
)

// type FaceSet struct {
// 	faceSetName  string `json:"facesetname"`
// 	faceSetID    string `json:"facesetid"`
// 	createTime   string `json:"createtime"`
// }

func InsertIntoFacedb(db *sql.DB, facesetName, faceID string, face []byte, imagebase64, name, age, address, imageaddress, imageurl string, createtime int64, similaryimageURL, similarity string, table string) error {
	stmt, err := db.Prepare(fmt.Sprintf("INSERT %s SET facesetname=?,faceid=?,face=?,imagebase64=?,name=?,age=?,address=?,imageaddress=?,imageurl=?,createtime=?,similaryimageURL=?,similarity=?", table))
	if err != nil {
		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(facesetName, faceID, face, imagebase64, name, age, address, imageaddress, imageurl, createtime, similaryimageURL, similarity)
	if err != nil {
		glog.Errorf("INSERT facedb err: %s", err.Error())
		// stmt.Close()
		return err
	}
	return nil
}

// func InsertIntoknowfaceinfo(db *sql.DB, facesetName, faceID string, face []byte, imagebase64, name, age, address, imageaddress, imageurl string, createtime int64) error {
// 	stmt, err := db.Prepare("INSERT knowfaceinfo SET facesetid=?,faceid=?,face=?,imagebase64=?,name=?,age=?,address=?,imageaddress=?,imageurl=?,createtime=?")
// 	if err != nil {
// 		glog.Errorf("Prepare INSERT faceinfo err: %s", err.Error())
// 		return err
// 	}
// 	_, err = stmt.Exec(facesetName, faceID, face, imagebase64, name, age, address, imageaddress, imageurl, createtime)
// 	if err != nil {
// 		glog.Errorf("INSERT facedb err: %s", err.Error())
// 		return err
// 	}
// 	return nil
// }

// func Listfaces() {
// 	rows, err := db.Query("select * from faceset where facesetname = ?", config.FaceSetName)
// 	if err != nil {
// 		db.Close()
// 		glog.Errorf("Query db err: %s", err.Error())
// 		os.Exit(1)
// 	}
// 	var face FaceSet
// 	facesets := make([]FaceSet, 0)
// 	for rows.Next() {
// 		var id int
// 		var facesetname interface{}
// 		err := rows.Scan(&id, &face.faceSetID, &facesetname, &face.createTime)
// 		if err != nil {
// 			db.Close()
// 			glog.Errorf("scan db err: %s", err.Error())
// 			os.Exit(1)
// 		}
// 		facesets = append(facesets, face)
// 	}
// }

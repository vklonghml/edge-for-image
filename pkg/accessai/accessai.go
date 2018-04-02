package accessai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	// "strings"
	"time"

	http_utils "edge-for-image/pkg/http"

	"github.com/golang/glog"
	"github.com/satori/go.uuid"

)

// Accessai define client to ad
type Accessai struct {
	HTTPClient *http.Client
}

// NewAccessai create accesai client
func NewAccessai() *Accessai {
	return &Accessai{
		http_utils.NewHTTPClient(),
	}
}

// FakeAddFace create face
func (ai *Accessai) FakeAddFace(urlStr, httpMethod string, body []byte) ([]byte, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		glog.Errorf("uuid generate error: %s", err.Error())
	}
	resp := []byte(fmt.Sprintf("{\"faceSetName\":\"faceset2\",\"faceID\":\"%s\",\"faceSetID\":\"e5919040-ae9e-4ad2-baac-4de04c46ffc4\",\"face\":{\"confidence\":\"95\",\"boundingBox\":{\"topLeftX\":\"0\",\"topLeftY\":\"0\",\"width\":\"100\",\"height\":\"100\"}}}", uid))
	return resp, nil
}

// FakeCreateFaceset create face set
func (ai *Accessai) FakeCreateFaceset(urlStr, httpMethod string, body []byte) ([]byte, error) {
	bdata := make(map[string]interface{})
	json.Unmarshal(body, &bdata)
	uid, err := uuid.NewV4()
	if err != nil {
		glog.Errorf("uuid generate error: %s", err.Error())
	}
	resp := []byte(fmt.Sprintf("{\"faceSetName\":\"%s\",\"faceSetID\":\"%s\",\"createDate\":\"%s\"}", bdata["faceSetName"], uid, time.Now()))
	return resp, nil
}

// FakeFaceSearch create face set
func (ai *Accessai) FakeFaceSearch(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp := []byte(fmt.Sprintf("{\"faceSetName\":\"faceset2\",\"faces\":[{\"externalImageID\":\"123-external\",\"faceID\":\"0b621d18-c59e-4c6e-9599-2b34fb3a0d45\",\"similarity\":\"75\"}]}"))
	return resp, nil
}

// FaceSearch create face set
func (ai *Accessai) FaceSearch(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp, err := access(urlStr, nil, body, len(body), httpMethod, ai.HTTPClient)
	return resp, err
}

// FaceDetect create face set
func (ai *Accessai) FaceDetect(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp, err := access(urlStr, nil, body, len(body), httpMethod, ai.HTTPClient)
	return resp, err
}

// CreateFaceset create face set
func (ai *Accessai) CreateFaceset(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp, err := access(urlStr, nil, body, len(body), httpMethod, ai.HTTPClient)
	return resp, err
}

// DeleteFaceset delete face set
func (ai *Accessai) DeleteFaceset(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp, err := access(urlStr, nil, body, len(body), httpMethod, ai.HTTPClient)
	return resp, err
}

// AddFace create face
func (ai *Accessai) AddFace(urlStr, httpMethod string, body []byte) ([]byte, error) {
	resp, err := access(urlStr, nil, body, len(body), httpMethod, ai.HTTPClient)
	return resp, err
}

// GetFace get face
func (ai *Accessai) GetFace(urlStr, httpMethod string,) ([]byte, error) {
	resp, err := access(urlStr, nil, []byte(""), 0, httpMethod, ai.HTTPClient)
	return resp, err
}

// DeleteFace delete face
func (ai *Accessai) DeleteFace(urlStr, httpMethod string,) ([]byte, error) {
	resp, err := access(urlStr, nil, []byte(""), 0, httpMethod, ai.HTTPClient)
	return resp, err
}

func access(URL string, headers map[string]string, content []byte, contentLength int, httpMethod string, httpclient *http.Client) ([]byte, error) {
	// glog.Infof("body: %s", content)
	var reqBody io.Reader
	var req *http.Request
	var err error
	if contentLength != 0 {
		reqBody = bytes.NewReader(content)
	} else {
		reqBody = nil
	}

	req, err = http_utils.BuildRequest(httpMethod, URL, reqBody, "")
	if err != nil {
		return nil, err
	}

	resp, err := http_utils.SendRequest(req, httpclient)
	if err != nil {
		glog.Errorf("error sending data to ai service %s", err.Error())
		return nil, fmt.Errorf("error sending data to ai service %s", err.Error())
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	glog.Infof("response from ai: %s, return code:%s", bodyString, string(resp.StatusCode))
	return bodyBytes, err
}

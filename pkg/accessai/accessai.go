package accessai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	http_utils "edge-for-image/pkg/http"

	"github.com/golang/glog"
	"edge-for-image/pkg"
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

// FaceDetect
func (ai *Accessai) FaceDetect(imageUrl string) (*FaceDetectResponse, error) {
	glog.Infof("FaceDetect: image url is: %s", imageUrl)
	req := &FaceDetectRequest{
		ImageUrl: "/" + pkg.Config0.OBSBucketName + "/" + imageUrl,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceDetectUrl(), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceDetect: return body is: %s.", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceDetectResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// FaceDetect
func (ai *Accessai) FaceDetectBase64(imageBase64 string) (*FaceDetectResponse, error) {
	req := &FaceDetectBase64Request{
		ImageBase64: imageBase64,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceDetectUrl(), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceDetect: return body is: %s.", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceDetectResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// FaceCompare
func (ai *Accessai) FaceCompare(image1Url, image2Url string) (*FaceCompareResponse, error) {
	glog.Infof("FaceCompare: image1 is: %s, image2 is %s.", image1Url, image2Url)
	req := &FaceCompareRequest{
		Image1Url: "/" + pkg.Config0.OBSBucketName + "/" + image1Url,
		Image2Url: "/" + pkg.Config0.OBSBucketName + "/" + image2Url,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceCompareUrl(), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceCompare: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceCompareResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// FaceCompareBase64
func (ai *Accessai) FaceCompareBase64(image1Base64, image2Base64 string) (*FaceCompareResponse, error) {
	req := &FaceCompareBase64Request{
		Image1Base64: image1Base64,
		Image2Base64: image2Base64,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceCompareUrl(), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceCompare: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceCompareResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// AddFace create face
func (ai *Accessai) AddFace(faceSetName, imageUrl string) (*AddFaceResponse, error) {
	glog.Infof("AddFace: faceSetName is: %s, imageUrl is %s", faceSetName, imageUrl)
	req := &AddFaceRequest{
		ImageUrl: "/" + pkg.Config0.OBSBucketName + "/" + imageUrl,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getAddFaceUrl(faceSetName), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("AddFace: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := AddFaceResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// AddFace create face base64
func (ai *Accessai) AddFaceBase64(faceSetName, imageBase64 string) (*AddFaceResponse, error) {
	glog.Infof("AddFace: faceSetName is: %s", faceSetName)
	req := &AddFaceBase64Request{
		ImageBase64: imageBase64,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getAddFaceUrl(faceSetName), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("AddFace: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := AddFaceResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// GetFace get face
func (ai *Accessai) GetFace(faceSetName, faceId string) (*GetFaceResponse, error) {
	glog.Infof("GetFace: faceSetName is: %s, faceId is %s", faceSetName, faceId)
	resp, err := access(getGetFaceUrl(faceSetName, faceId), nil, []byte(""), 0, http.MethodGet, ai.HTTPClient)
	glog.Infof("GetFace: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := GetFaceResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// DeleteFace delete face
func (ai *Accessai) DeleteFace(faceSetName, faceId string) (*DeleteFaceResponse, error) {
	glog.Infof("DeleteFace: faceSetName is: %s, faceId is %s", faceSetName, faceId)
	resp, err := access(getDeleteFaceUrl(faceSetName, faceId), nil, []byte(""), 0, http.MethodDelete, ai.HTTPClient)
	glog.Infof("DeleteFace: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := DeleteFaceResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// CreateFaceset create face set
func (ai *Accessai) CreateFaceset(faceSetName string, faceSetCapacity int64) (*CreateFacesetResponse, error) {
	glog.Infof("CreateFaceset: faceSetName is: %s, faceSetCapacity is %d", faceSetName, faceSetCapacity)
	req := &CreateFacesetRequest{
		FaceSetName:     faceSetName,
		FaceSetCapacity: faceSetCapacity,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getCreateFaceSetUrl(), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("CreateFaceset: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := CreateFacesetResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

func (ai *Accessai) ListFaceset(faceSetName string) (*ListFacesetResponse, error) {
	glog.Infof("ListFaceset: faceSetName is: %s", faceSetName)
	resp, err := access(getListFaceSetUrl(), nil, []byte(""), 0, http.MethodPost, ai.HTTPClient)
	glog.Infof("CreateFaceset: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := ListFacesetResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// DeleteFaceset delete face set
func (ai *Accessai) DeleteFaceset(faceSetName string) (*DeleteFacesetResponse, error) {
	glog.Infof("DeleteFaceset: faceSetName is: %s", faceSetName)
	resp, err := access(getDeleteFaceSetUrl(faceSetName), nil, []byte(""), 0, http.MethodDelete, ai.HTTPClient)
	glog.Infof("DeleteFaceset: return body is: %s, ", string(resp))

	if err != nil {
		return nil, err
	}
	response := DeleteFacesetResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// FaceSearch create face set
func (ai *Accessai) FaceSearch(faceSetName, imageUrl string) (*FaceSearchResponse, error) {
	glog.Infof("FaceSearch: faceSetName is: %s, imageUrl is: %s", faceSetName, imageUrl)
	req := &FaceSearchRequest{
		ImageUrl: "/" + pkg.Config0.OBSBucketName + "/" + imageUrl,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceSearchUrl(faceSetName), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceSearch: return body is: %s.", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceSearchResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

// FaceSearch create face set
func (ai *Accessai) FaceSearchBase64(faceSetName, imageBase64 string) (*FaceSearchResponse, error) {
	glog.Infof("FaceSearch: faceSetName is: %s", faceSetName)
	req := &FaceSearchBase64Request{
		ImageBase64: imageBase64,
	}
	body, _ := json.Marshal(req)
	resp, err := access(getFaceSearchUrl(faceSetName), nil, body, len(body), http.MethodPost, ai.HTTPClient)
	glog.Infof("FaceSearch: return body is: %s.", string(resp))

	if err != nil {
		return nil, err
	}
	response := FaceSearchResponse{}
	err = json.Unmarshal(resp, &response)
	return &response, err
}

func access(URL string, headers map[string]string, content []byte, contentLength int, httpMethod string, httpclient *http.Client) ([]byte, error) {
	//var body string
	//if len(string(content)) >= 100 {
	//	body = string(content)[0:100]
	//} else {
	//	body = string(content)
	//}
	//glog.Infof("url is %v, body is %v.", URL, body)
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

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Auth-Token", TOKEN)

	resp, err := http_utils.SendRequest(req, httpclient)
	if err != nil {
		glog.Errorf("error sending data to ai service %s", err.Error())
		return nil, fmt.Errorf("error sending data to ai service %s", err.Error())
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return bodyBytes, fmt.Errorf(string(bodyBytes))
	}
	return bodyBytes, err
}

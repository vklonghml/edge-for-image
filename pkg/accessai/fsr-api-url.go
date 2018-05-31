package accessai

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// const TOKEN = "XXXX"

var (
	tokenTemplate = `{
		"auth": {
		  "identity": {
			"methods": [
			  "password"
			],
			"password": {
			  "user": {
				"name": "{{ .Name }}",
				"password": "{{ .Password }}",
				"domain": {
				  "name": "{{ .Domain }}"
				}
			  }
			}
		  },
		  "scope": {
			"project": {
			  "name": "{{ .Project }}"
			}
		  }
		}
	  }`
	TOKEN     string
	ProjectID string
)

var (
	//frsUrl    = pkg.Config0.Aiurl
	frsUrl = "https://frs.cn-north-1.myhuaweicloud.com/v1/"
)

// IAMClient type
type IAMClient struct {
	httpClient *http.Client
	url        string
	name       string
	password   string
	project    string
	domain     string
}

// NewIAMClient create iam client
func NewIAMClient(url, name, password, project, domain string) *IAMClient {
	tp := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Timeout: time.Second * 10, Transport: tp}
	password = strings.TrimSpace(password)
	return &IAMClient{
		httpClient: client,
		url:        url,
		name:       name,
		password:   password,
		project:    project,
		domain:     domain,
	}
}

// GetToken get token
func (c *IAMClient) GetToken() (string, string, error) {
	var buffer bytes.Buffer
	data := map[string]string{
		"Name":     c.name,
		"Password": c.password,
		"Domain":   c.domain,
		"Project":  c.project,
	}
	template.Must(template.New("token").Parse(tokenTemplate)).Execute(&buffer, data)
	req, err := http.NewRequest(http.MethodPost, c.url, &buffer)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-type", "application/json;charset=utf8")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("failed to get token, %d", resp.StatusCode)
	}
	TOKEN = resp.Header.Get("X-Subject-Token")
	if TOKEN == "" {
		return "", "", fmt.Errorf("empty token")
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	body := make(map[string]interface{})
	err = json.Unmarshal(content, &body)
	if err != nil {
		return "", "", err
	}
	tokenMap := body["token"].(map[string]interface{})
	projectMap := tokenMap["project"].(map[string]interface{})
	ProjectID = projectMap["id"].(string)
	return TOKEN, ProjectID, nil
}

func getFaceDetectUrl() string {
	return fmt.Sprintf("%s%s/face-detect", frsUrl, ProjectID)
}

func getFaceCompareUrl() string {
	return fmt.Sprintf("%s%s/face-compare", frsUrl, ProjectID)
}

func getAddFaceUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces", frsUrl, ProjectID, faceSetName)
}

func getGetFaceUrl(faceSetName, faceId string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces?face_id=%s", frsUrl, ProjectID, faceSetName, faceId)
}

func getDeleteFaceUrl(faceSetName, faceId string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces?face_id=%s", frsUrl, ProjectID, faceSetName, faceId)
}

func getCreateFaceSetUrl() string {
	return fmt.Sprintf("%s%s/face-sets", frsUrl, ProjectID)
}

func getListFaceSetUrl() string {
	return fmt.Sprintf("%s%s/face-sets", frsUrl, ProjectID)
}

func getGetFaceSetUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s", frsUrl, ProjectID, faceSetName)
}

func getDeleteFaceSetUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s", frsUrl, ProjectID, faceSetName)
}

func getFaceSearchUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/search", frsUrl, ProjectID, faceSetName)
}

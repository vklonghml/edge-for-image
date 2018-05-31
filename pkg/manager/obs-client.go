package manager

import (
	"encoding/base64"
	"github.com/golang/glog"
	"errors"
	"edge-for-image/pkg/obs"
	"edge-for-image/pkg"
	"bytes"
	"fmt"
)

func (m *Manager) UploadImageToObs(inputKey, imageBase64 string) error {
	imageDecode, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		glog.Error("DecodeString err")
		return errors.New("DecodeString image err")
	}

	input := &obs.PutObjectInput{}
	input.Bucket = pkg.Config0.OBSBucketName
	input.Key = inputKey
	input.Body = bytes.NewReader(imageDecode)

	output, err := m.ObsClient.PutObject(input)
	if obsError, ok := err.(obs.ObsError); ok {
		glog.Errorf("Code: %s", obsError.Code)
		glog.Errorf("Message: %s", obsError.Message)
		return fmt.Errorf("put image %s to obs failed.", input.Key)
	}
	glog.Infof("Upload image success, Key: %s, RequestId: %s, ETag: %s", inputKey, output.RequestId, output.ETag)

	return nil
}

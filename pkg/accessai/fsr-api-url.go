package accessai

import "fmt"

const TOKEN  = "XXXX"

var (
	projectId = "e73c384a7c5f475886afc5e7d5fd09fd"
	frsUrl    = "https://frs.cn-north-1.myhuaweicloud.com/v1/"
)

func getFaceDetectUrl() string {
	return fmt.Sprintf("%s%s/face-detect", frsUrl, projectId)
}

func getFaceCompareUrl() string {
	return fmt.Sprintf("%s%s/face-compare", frsUrl, projectId)
}

func getAddFaceUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces", frsUrl, projectId, faceSetName)
}

func getGetFaceUrl(faceSetName, faceId string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces?face_id=%s", frsUrl, projectId, faceSetName, faceId)
}

func getDeleteFaceUrl(faceSetName, faceId string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/faces?face_id=%s", frsUrl, projectId, faceSetName, faceId)
}

func getCreateFaceSetUrl() string {
	return fmt.Sprintf("%s%s/face-sets", frsUrl, projectId)
}

func getListFaceSetUrl() string {
	return fmt.Sprintf("%s%s/face-sets", frsUrl, projectId)
}

func getGetFaceSetUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s", frsUrl, projectId, faceSetName)
}

func getDeleteFaceSetUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s", frsUrl, projectId, faceSetName)
}

func getFaceSearchUrl(faceSetName string) string {
	return fmt.Sprintf("%s%s/face-sets/%s/search", frsUrl, projectId, faceSetName)
}

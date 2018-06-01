package accessai

type ErrorResponse struct {
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type FaceDetectRequest struct {
	ImageUrl string `json:"image_url"`
}

type FaceDetectBase64Request struct {
	ImageBase64 string `json:"image_base64"`
}

type faces struct {
	BoundingBox     boundingBox `json:"bounding_box"`
	Similarity      float64     `json:"similarity"`
	ExternalImageId string      `json:"external_image_id"`
	FaceId          string      `json:"face_id"`
}

type boundingBox struct {
	TopLeftX int32 `json:"top_left_x"`
	TopLeftY int32 `json:"top_left_y"`
	Width    int32 `json:"width"`
	Height   int32 `json:"height"`
}

type FaceDetectResponse struct {
	Faces []faces `json:"faces"`
}

type FaceCompareRequest struct {
	Image1Url string `json:"image1_url"`
	Image2Url string `json:"image2_url"`
}

type FaceCompareBase64Request struct {
	Image1Base64 string `json:"image1_base64"`
	Image2Base64 string `json:"image2_base64"`
}

type imageFace struct {
	BoundingBox boundingBox `json:"bounding_box"`
}
type FaceCompareResponse struct {
	Similarity float64   `json:"similarity"`
	Image1Face imageFace `json:"image1_face"`
	Image2Face imageFace `json:"image2_face"`
}

type AddFaceRequest struct {
	ImageUrl string `json:"image_url"`
	//ExternalImageId string `json:"external_image_id"`
}

type AddFaceBase64Request struct {
	ImageBase64 string `json:"image_base64"`
	//ExternalImageId string `json:"external_image_id"`
}

type AddFaceResponse struct {
	FaceSetId   string  `json:"face_set_id"`
	FaceSetName string  `json:"face_set_name"`
	Faces       []faces `json:"faces"`
}

type GetFaceResponse struct {
	FaceSetId   string  `json:"face_set_id"`
	FaceSetName string  `json:"face_set_name"`
	Faces       []faces `json:"faces"`
}

type DeleteFaceResponse struct {
	FaceIds         string `json:"face_ids"`
	ExternalImageId string `json:"external_image_id"`
	FaceSetId       string `json:"face_set_id"`
	FaceSetName     string `json:"face_set_name"`
}

type CreateFacesetRequest struct {
	FaceSetName     string `json:"face_set_name"`
	FaceSetCapacity int64  `json:"face_set_capacity"`
}

type faceSetInfo struct {
	FaceNumber      int64  `json:"face_number"`
	FaceSetId       string `json:"face_set_id"`
	FaceSetName     string `json:"face_set_name"`
	CreateDate      string `json:"create_date"`
	FaceSetCapacity int64  `json:"face_set_capacity"`
}

type CreateFacesetResponse struct {
	FaceSetInfo faceSetInfo `json:"face_set_info"`
}

type ListFacesetResponse struct {
	FaceSetsInfo []faceSetInfo `json:"face_sets_info"`
}

type GetFacesetResponse struct {
	FaceSetInfo faceSetInfo `json:"face_set_info"`
}

type DeleteFacesetResponse struct {
	FaceSetName string `json:"face_set_name"`
}

type FaceSearchRequest struct {
	ImageUrl string `json:"image_url"`
	TopN     int32  `json:"top_n"`
}

type FaceSearchBase64Request struct {
	ImageBase64 string `json:"image_base64"`
}

type FaceSearchResponse struct {
	Faces []faces `json:"faces"`
}

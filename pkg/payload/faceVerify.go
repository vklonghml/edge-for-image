package payload

type FaceVerifyRequest struct {
	Image1URL string `json:"image1Url"`
	Image2Url string `json:"image2Url"`
}

type FaceVerifyResponse struct {
	Similarity string    `json:"similarity"`
	Image1Face imageFace `json:"image1Face"`
	Image2Face imageFace `json:"image2Face"`
}

type imageFace struct {
	Confidence  string      `json:"confidence"`
	BoundingBox boundingBox `json:"boundingBox"`
}

type boundingBox struct {
	TopLeftX int32 `json:"topLeftX"`
	TopLeftY int32 `json:"topLeftY"`
	Width    int32 `json:"width"`
	Height   int32 `json:"height"`
}

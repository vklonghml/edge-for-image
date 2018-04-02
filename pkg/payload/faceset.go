package payload

type FacesetRequest struct {
	Faceset Faceset `json:"faceset"`
}

type Faceset struct {
	Name 	string `json:"name"`
}

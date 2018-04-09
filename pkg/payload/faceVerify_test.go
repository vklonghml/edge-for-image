package payload

import (
	"encoding/json"
	"fmt"
	"testing"
)

const BODY1 = "{\"image1Face\":{\"boundingBox\":{\"height\":258,\"topLeftX\":113,\"topLeftY\":220,\"width\":258},\"confidence\":\"1.0\"},\"image2Face\":{\"boundingBox\":{\"height\":258,\"topLeftX\":113,\"topLeftY\":220,\"width\":258},\"confidence\":\"1.0\"},\"similarity\":\"1.0\"}"

func TestFaceVerifyResponseFromJson(t *testing.T) {
	py := &FaceVerifyResponse{}
	json.Unmarshal([]byte(BODY1), &py)
	fmt.Println(py.Similarity)
}

func TestFaceVerifyResponseToJson(t *testing.T) {
	resp := FaceVerifyResponse{
		Similarity: "1.0",
	}
	bdate, _ := json.Marshal(resp)
	fmt.Println(string(bdate))
}

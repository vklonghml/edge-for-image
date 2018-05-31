package accessai

import (
	"testing"
	"encoding/json"
	"fmt"
	"log"
)

const (
	ErrorResponseBody      = "{\"error_code\":\"FRS.0601\",\"error_msg\":\"Can not detect face.\"}"
	FaceDetectResponseBody = "{\"faces\":[{\"bounding_box\":{\"top_left_x\":200,\"top_left_y\":100,\"width\":120,\"height\":120}}]}"
)

func TestErrorResponseToJson(t *testing.T) {
	er := ErrorResponse{
		ErrorCode: "FRS.0601",
		ErrorMsg:  "Can not detect face.",
	}

	plJson, _ := json.Marshal(er)
	assertEqual(t, ErrorResponseBody, string(plJson))
}

func TestFaceDetectResponseToJson(t *testing.T) {
	bb := &boundingBox{
		Width:    120,
		Height:   120,
		TopLeftY: 100,
		TopLeftX: 200,
	}

	bbarray := make([]boundingBox, 1)
	bbarray[0] = *bb
	faces0 := &faces{
		BoundingBox: *bb,
	}

	faces1 := make([]faces, 1)
	faces1[0] = *faces0

	py := &FaceDetectResponse{
		Faces: faces1,
	}

	plJson, _ := json.Marshal(py)
	assertEqual(t, FaceDetectResponseBody, string(plJson))
}

func TestFaceDetectResponseFromJson(t *testing.T) {
	py := &FaceDetectResponse{}
	err := json.Unmarshal([]byte(FaceDetectResponseBody), &py)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v", py.Faces[0].BoundingBox.TopLeftX)
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

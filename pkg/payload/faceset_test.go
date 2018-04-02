package payload

import (
	"encoding/json"
	"fmt"
	"testing"
)

const BODY = "{\"faceset\":{\"name\":\"pujie\"}}"

func TestFacesetRequestToJson(t *testing.T) {
	py := &FacesetRequest{
		Faceset: Faceset{
			Name: "pujie",
		},
	}
	plJson, _ := json.Marshal(py)
	assertEqual(t, BODY, string(plJson))
}

func TestFacesetRequestFromJson(t *testing.T) {
	py := &FacesetRequest{}
	json.Unmarshal([]byte(BODY), &py)
	fmt.Println(py.Faceset.Name)
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

package model

type PicSample struct {
	Id           string
	UploadTime   int64
	Similarity   map[string]int32
	MostSimilar  string
	ImageBase64  string
	ImageUrl     string
	ImageAddress string
}

type SimilaryRelation struct {
	From     *PicSample
	To       *PicSample
	Similary int32
}

//key is faceset_name
//type UploadPortal map[string]Caches

type Caches struct {
	DetectCache       []PicSample
	RegisterCache     []PicSample
	TempRegisterCache []PicSample
	TempDetectCache   []PicSample
}

type SRList []SimilaryRelation

type SRMatrix []SRList

type LastSaveMap map[*PicSample]int64

type SRIdMap map[string]int32

type SRSingleMap map[*PicSample]int32

type SRDoubleMap map[*PicSample]map[*PicSample]int32

const CACHE_SUFFIX = "_cache"

const PIC_TIME_OUT_SEC = 10

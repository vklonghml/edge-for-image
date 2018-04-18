package model

type PicSample struct {
	Id            string
	UploadTime    int64
	Similarity    map[string]int32
	MostSimilar   int32
	MostSimilarId string
	ImageBase64   string
	ImageUrl      string
	ImageAddress  string
	Face          []byte
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

type SRIdMap map[string]int32

type SRSingleMap map[*PicSample]int32

type SRDoubleMap map[*PicSample]map[*PicSample]int32

const CACHE_SUFFIX = "_cache"

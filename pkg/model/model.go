package model

import "container/list"

type PicSample struct {
	Id          string
	UploadTime  int64
	Similarity  map[string]int32
	MostSimilar string
	ImageBase64 string
}

type SimilaryRelation struct {
	From       *PicSample
	To         *PicSample
	Similary   int32
	UploadTime int64
}

type UploadPortal struct {
	DetectCache   *list.List
	RegisterCache *list.List
	TempRegisterCache	*list.List
	TempDetectCache		*list.List
}

type SRList	[]SimilaryRelation

type SRMatrix		[]SRList

type LastSaveMap	map[*PicSample]int64

type SRFromMap 		map[*PicSample]int32

type SRToMap		map[*PicSample]int32

type SRMap 			map[*PicSample]map[*PicSample]int32

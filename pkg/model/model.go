package model

import "container/list"

type PicSample struct {
	Id          string
	UploadTime  int64
	Similarity  map[string]int64
	MostSimilar string
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
}



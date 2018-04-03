package model

import "container/list"

type PicSample struct {
	Id          string
	UploadTime  uint64
	Similarity  map[string]uint32
	MostSimilar string
}

type SimilaryRelation struct {
	From       PicSample
	To         PicSample
	Similary   uint32
	UploadTime uint64
}

type UploadPortal struct {
	DetectCache   *list.List
	RegisterCache *list.List
}


func (up *UploadPortal) caculateSimilarity(sample *PicSample) {

}

func (up *UploadPortal) caculateMostSimilarity(sample *PicSample) {

}




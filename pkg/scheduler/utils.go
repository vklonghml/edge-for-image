package scheduler

import "edge-for-image/pkg/model"

func getCacheFacesetName(facesetname string) string {
	return facesetname + model.CACHE_SUFFIX
}

package scheduler

import (
	"edge-for-image/pkg/manager"
	"github.com/golang/glog"
	"time"
)

func (s *Scheduler) LoopHandleRegistCache(m *manager.Manager, facesetName string) {
	for {

		length := len(m.RegistCache[facesetName])
		if length == 0 {
			time.Sleep(time.Second * time.Duration(m.CustConfig.RegistPeriodSec))
		} else if length >= m.CustConfig.RegistCacheSize {
			glog.Infof("RegistCache Loop : %s -----------------------------------------------------------", facesetName)
			s.ScheduleRegisterCacheUse1v1(facesetName, m);
		} else if length > 0 && length < m.CustConfig.RegistCacheSize {
			glog.Infof("RegistCache Loop : %s -----------------------------------------------------------", facesetName)
			time.Sleep(time.Second * time.Duration(m.CustConfig.RegistPeriodSec))
			s.ScheduleRegisterCacheUse1v1(facesetName, m);
		}
		//glog.Infof("RegistCache Loop : %s ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^", facesetName)
	}
}

func (s *Scheduler) LoopHandleDetectCache(m *manager.Manager, facesetName string) {
	for {

		length := len(m.DetectCache[facesetName])
		if length == 0 {
			time.Sleep(time.Second * time.Duration(m.CustConfig.DetectPeriodSec))
		} else if length >= m.CustConfig.DetectCacheSize {
			glog.Infof("DetectCache Loop : %s ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^", facesetName)
			s.ScheduleDetectCacheUse1v1(facesetName, m);
		} else if length > 0 && length < m.CustConfig.DetectCacheSize {
			glog.Infof("DetectCache Loop : %s ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^", facesetName)
			time.Sleep(time.Second * time.Duration(m.CustConfig.DetectPeriodSec))
			s.ScheduleDetectCacheUse1v1(facesetName, m);
		}
		//glog.Infof("DetectCache Loop : %s ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^", facesetName)
	}
}

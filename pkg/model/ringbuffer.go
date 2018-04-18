package model

import (
	"log"
	"github.com/golang/glog"
)

type Queen struct {
	Length   int64
	Capacity int64
	Head     int64
	Tail     int64
	Data     []interface{}
}

func MakeQueen(length int64) *Queen {
	var q = &Queen{Length: length, Data: make([]interface{}, length),}
	return q
}

func (t *Queen) IsEmpty() bool {
	return t.Capacity == 0
}

func (t *Queen) IsFull() bool {
	return t.Capacity == t.Length
}

func (t *Queen) Append(element interface{}) bool {
	if t.IsFull() {
		log.Println("queen is full.")
		return false
	}
	glog.Infof("Tail is %d\n", t.Tail)
	t.Data[t.Tail%t.Length] = element
	t.Tail++
	t.Capacity++
	return true
}

func (t *Queen) OutElement() interface{} {
	if t.IsEmpty() {
		log.Println("queen is empty.")
	}
	defer func() {
		t.Capacity--
		t.Head++
	}()
	glog.Infof("Head is %d\n", t.Head)
	return t.Data[t.Head%t.Length]
}

func (t *Queen) Each(fn func(node interface{})) {
	for i := t.Head; i < t.Head+t.Capacity; i++ {
		fn(t.Data[i%t.Length])
	}
}

func (t *Queen) Clear() bool {
	t.Capacity = 0
	t.Head = 0
	t.Tail = 0
	t.Data = make([]interface{}, t.Length)
	return true
}

package model

import "testing"

func TestRingBuffer(t *testing.T) {
	queue := MakeQueen(2)
	queue.Append("a")
	queue.Append("a")
	queue.OutElement()
	queue.OutElement()
	queue.Append("a")
	queue.Append("a")
	queue.Append("a")
}

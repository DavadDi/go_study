package util

import (
	"testing"
)

func TestNewSet(t *testing.T) {
	s := NewSet()

	str := "id"
	s.Add(str)

	if s.Size() != 1 {
		t.Errorf("Set Add Check failed")
	}

	if !s.Exist(str) {
		t.Errorf("Set Exist Check failed")
	}
}

func TestNewSetInit(t *testing.T) {
	strs := []string{"id1", "id2", "id3"}
	s := NewSet("id1", "id2", "id3")

	if s.Size() != 3 {
		t.Errorf("Set Add Check failed")
	}

	for _, v := range strs {
		if !s.Exist(v) {
			t.Errorf("Set Exist Check failed fpr [%s]", v)
		}
	}

	// should return false
	if s.Add("id1") {
		t.Errorf("Add exist item should return false.")
	}

	s.Remove("id1")

	if s.Size() != 2 {
		t.Errorf("Set Remove Check failed")
	}
}

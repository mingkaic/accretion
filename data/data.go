package data

import (
	"crypto/sha1"
)

type (
	Leaf struct {
		Id          string      `json:"id"`
		Annotations []string    `json:"-"`
		Label       string      `json:"-"`
		Shape       []uint64    `json:"-"`
		Data        []float64   `json:"-"`
		Sinfo       *SparseInfo `json:"-"`
	}

	Func struct {
		Id          string   `json:"id"`
		Args        []string `json:"args"`
		Annotations []string `json:"-"`
		Opname      string   `json:"-"`
		Runtime     uint64   `json:"-"`
	}

	SparseInfo struct {
		NonZeros     int     `json:"-"`
		Indices      []int32 `json:"-"`
		OuterIndices []int64 `json:"-"`
	}

	Annotation struct {
		Id    string `json:"-"`
		Key   string `json:"-"`
		Value string `json:"-"`
	}
)

func NewAnnotation(key, val string) *Annotation {
	kh := sha1.Sum([]byte(key))
	vh := sha1.Sum([]byte(val))
	id := sha1.Sum(append(kh[:], vh[:]...))
	return &Annotation{
		Id:    string(id[:]),
		Key:   key,
		Value: val,
	}
}

func (l *Leaf) ToString() string {
	return l.Id
}

func (f *Func) ToString() string {
	return f.Id
}

func (a *Annotation) ToString() string {
	return a.Id
}

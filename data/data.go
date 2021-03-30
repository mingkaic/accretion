package data

import (
	"crypto/sha1"
)

type (
	TenncorNode struct {
		Id          string      `json:"id"`
		Label       string      `json:"label"`
		Shape       []uint64    `json:"shape"`
		Runtime     uint64      `json:"runtime",omitempty`
		Annotations []string    `json:"-"`
		Args        []string    `json:"-"`
		Data        []float64   `json:"-"`
		Sinfo       *SparseInfo `json:"-"`
	}

	Annotation struct {
		Id    string `json:"aid"`
		Key   string `json:"key"`
		Value string `json:"val"`
	}

	SparseInfo struct {
		Indices      []int32 `json:"-"`
		OuterIndices []int64 `json:"-"`
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

func (tn *TenncorNode) ToString() string {
	return tn.Id
}

func (a *Annotation) ToString() string {
	return a.Id
}

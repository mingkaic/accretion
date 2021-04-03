package data

import (
	"crypto/sha1"
	"fmt"
)

type (
	TenncorNode struct {
		Uid         string         `json:"uid"`
		ProfileId   string         `json:"profile_id"`
		Id          string         `json:"id"`
		Label       string         `json:"label"`
		Shape       []uint64       `json:"shape"`
		Runtime     uint64         `json:"runtime,omitempty"`
		Args        []*TenncorNode `json:"arg,omitempty"`
		Annotations []*Annotation  `json:"-"`
		Data        []float64      `json:"-"`
		Sinfo       *SparseInfo    `json:"-"`
		ArgIds      []string       `json:"-"`
	}

	Annotation struct {
		Uid   string `json:"uid"`
		Id    string `json:"-"`
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
		Uid:   fmt.Sprintf("_:%s", id[:]),
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

package models

import (
	"hash/fnv"
	"time"
)

type Instance struct {
	ID           string
	IPAddress    string
	LastUsedTime time.Time
	CountOfLoads int
}

type Request struct {
	ID              string
	Payload         []byte
	ResponseChannel chan []byte
	URL             string
	Method          string
}

func (r *Request) HashCode() uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(r.ID))
	hasher.Write(r.Payload)
	hasher.Write([]byte(r.URL))
	hasher.Write([]byte(r.Method))
	return hasher.Sum64()
}

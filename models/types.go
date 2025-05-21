package models

import (
	"time"
)

type Instance struct {
	ID           string
	IPAddress    string
	LastUsedTime time.Time
	CountOfLoads int
}

type Request struct {
	Payload         []byte
	ResponseChannel chan []byte
	URL             string
	Method          string
}

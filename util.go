package main

import (
	"crypto/tls"
	"hash/crc32"
	"math/rand"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// getUUID generate a uuid v4
func getUUID() string {
	return uuid.New().String()
}

// randStringBytes generate a string with length n
func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// getHostId generate a tens digit as hostUid
func getHostId() int {
	customHash := func(s string) int {
		v := int(crc32.ChecksumIEEE([]byte(s)))
		if v >= 0 {
			return v
		}
		if -v >= 0 {
			return -v
		}
		// v == MinInt
		return 0
	}
	return customHash(getUUID())
}

// generateUserName generate a user name
func generateUserName(length int) string {
	return randStringBytes(length)
}

type poolData struct{ buf []byte }

// slice pool
// modify from https://github.com/golang/go/blob/c5c1d069da73a5e74bd2139ef1c7c14659915acd/src/net/http/h2_bundle.go#L1032
// var (
// 	dataChunkSizeClasses = []int{
// 		1 << 2,
// 		2 << 2,
// 		4 << 2,
// 		8 << 2,
// 		16 << 2,
// 		32 << 2,
// 		64 << 2,
// 	}
// 	dataChunkPools = [...]sync.Pool{
// 		{New: func() interface{} { return make([]byte, 1<<2) }},
// 		{New: func() interface{} { return make([]byte, 2<<2) }},
// 		{New: func() interface{} { return make([]byte, 4<<2) }},
// 		{New: func() interface{} { return make([]byte, 8<<2) }},
// 		{New: func() interface{} { return make([]byte, 16<<2) }},
// 		{New: func() interface{} { return make([]byte, 32<<2) }},
// 		{New: func() interface{} { return make([]byte, 64<<2) }},
// 	}
// )

// func getDataBufferChunk(size int64) []byte {
// 	i := 0
// 	for ; i < len(dataChunkSizeClasses)-1; i++ {
// 		if size <= int64(dataChunkSizeClasses[i]) {
// 			break
// 		}
// 	}
// 	p := dataChunkPools[i].Get().([]byte)
// 	p = p[:0]
// 	return p
// }

// func putDataBufferChunk(p []byte) {
// 	for i, n := range dataChunkSizeClasses {
// 		if cap(p) == n {
// 			dataChunkPools[i].Put(p)
// 			return
// 		}
// 	}
// 	log.Fatalf("unexpected buffer len=%v", len(p))
// }

// http transport pool

var totalTransport int = 0
var (
	transportPool = sync.Pool{New: func() interface{} {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		totalTransport += 1
		return tr
	}}
)

func getTransport() *http.Transport {
	return transportPool.Get().(*http.Transport)
}

func putTransport(t *http.Transport) {
	transportPool.Put(t)
}

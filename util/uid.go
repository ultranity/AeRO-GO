package util

import (
	"math/rand"
	"time"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base62      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	idxBits     = 6              // 6 bits to represent a letter index
	idxMask     = 1<<idxBits - 1 // All 1-bits, as many as letterIdxBits
	idxMax      = 63 / idxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func NewRandStrSource(seed int64) {
	src = rand.NewSource(seed)
}

// RandStringBytesMaskImprSrc 生成随机字符串
func RandStr(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), idxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), idxMax
		}
		if idx := int(cache & idxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= idxBits
		remain--
	}

	return string(b)
}

var buf8 = make([]byte, 8)

// warn: 由于采用取余数的方式，会导致生成的字符串不够随机(概率为1/62^10)
func RandStr8Base62() string {
	remain := 2
	for i, cache := 7, src.Int63(); i >= 0; {
		idx := int(cache & idxMask)
		if idx >= len(base62) {
			if remain > 0 {
				remain--
				cache >>= idxBits
				continue
			} else {
				idx = idx % len(base62)
			}
		}
		buf8[i] = base62[idx]
		i--
		cache >>= idxBits
	}
	return string(buf8)
}

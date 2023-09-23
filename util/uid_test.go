package util_test

import (
	"AeRO/proxy/util"
	"strings"
	"testing"
)

const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func TestUIDRandStr8(t *testing.T) {
	count := make([]int, 62)
	for i := 0; i < 10; i++ {
		str := util.RandStr8Base62()
		// count occurance of char in str
		for _, c := range str {
			count[strings.Index(base62, string(c))]++
		}
	}
	t.Logf("count:%v\n", count)
}

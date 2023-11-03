package netstat_test

import (
	"AeRO/proxy/netstat"
	"testing"
)

func BenchmarkGetAllSocksToPort(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		socks, _ := netstat.GetAllSocks(netstat.NoopFilter)
		_ = netstat.ToPorts(socks)
	}
	b.StopTimer()
}

func BenchmarkGetAllPort(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = netstat.GetAllPorts(netstat.NoopFilter)
	}
	b.StopTimer()
}

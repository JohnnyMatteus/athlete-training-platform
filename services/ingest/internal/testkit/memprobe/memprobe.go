// Package memprobe mede o pico de heap de um trecho de código, para os testes de memória constante.
package memprobe

import (
	"runtime"
	"time"
)

// Peak executa fn e devolve o maior crescimento de HeapAlloc sobre o valor inicial (após GC).
func Peak(fn func() error) (uint64, error) {
	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	stop := make(chan struct{})
	result := make(chan uint64)
	go func() {
		var peak uint64
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				result <- peak
				return
			case <-tick.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > peak {
					peak = m.HeapAlloc
				}
			}
		}
	}()
	err := fn()
	close(stop)
	peak := <-result
	if peak < base.HeapAlloc {
		return 0, err
	}
	return peak - base.HeapAlloc, err
}

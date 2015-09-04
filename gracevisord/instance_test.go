package main

import (
	"sync"
	"testing"
)

func BenchmarkServe(b *testing.B) {
	inst := &Instance{}
	inst.connWg = &sync.WaitGroup{}

	for i := 0; i < b.N; i++ {
		inst.Serve()
		inst.Done()
	}
}

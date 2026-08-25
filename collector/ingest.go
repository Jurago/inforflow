package main

import (
	"log"
	"runtime"
	"sync/atomic"
)

var (
	ingestCh      chan RawFlow
	ingestWorkers int32
)

func ingestQueueLen() int {
	if ingestCh == nil {
		return 0
	}
	return len(ingestCh)
}

func ingestWorkerCount() int {
	return int(atomic.LoadInt32(&ingestWorkers))
}

// startIngestPipeline separa decode UDP (listener) da classificação/agregação (workers).
func startIngestPipeline() {
	n := GetConfig().IngestWorkers
	if n <= 0 {
		n = runtime.NumCPU()
		if n < 2 {
			n = 2
		}
		if n > 8 {
			n = 8
		}
	}
	ingestCh = make(chan RawFlow, 131072)
	atomic.StoreInt32(&ingestWorkers, int32(n))
	for i := 0; i < n; i++ {
		go func() {
			for f := range ingestCh {
				ingestRaw(f)
			}
		}()
	}
	log.Printf("ingest: %d workers · fila %d", n, cap(ingestCh))
	go StartNetFlowListener(func(f RawFlow) {
		ingestCh <- f
	})
}

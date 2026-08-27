package main

import (
	"log"
	"runtime"
	"sync/atomic"
)

var (
	ingestCh      chan RawFlow
	storeCh       chan FlowRecord
	ingestWorkers int32
)

func ingestQueueLen() int {
	n := 0
	if ingestCh != nil {
		n += len(ingestCh)
	}
	if storeCh != nil {
		n += len(storeCh)
	}
	return n
}

func ingestWorkerCount() int {
	return int(atomic.LoadInt32(&ingestWorkers))
}

// startIngestPipeline: decode UDP → classify (N workers) → single AddFlow writer.
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
	storeCh = make(chan FlowRecord, 65536)
	atomic.StoreInt32(&ingestWorkers, int32(n))

	for i := 0; i < n; i++ {
		go func() {
			for raw := range ingestCh {
				if f, ok := classifyRaw(raw); ok {
					storeCh <- f
				}
			}
		}()
	}
	go func() {
		for f := range storeCh {
			store.AddFlow(f)
		}
	}()

	log.Printf("ingest: %d classify workers · rawQ %d · storeQ %d · single writer", n, cap(ingestCh), cap(storeCh))
	go StartNetFlowListener(func(f RawFlow) {
		ingestCh <- f
	})
}

package engine

import (
	"context"
	"sync"
)

const SizeWorkers = 4

func InParallel(ctx context.Context, count int, work func(index int)) error {
	if count < 1 {
		return ctx.Err()
	}

	workers := SizeWorkers
	if workers > count {
		workers = count
	}

	next := make(chan int)
	var waiting sync.WaitGroup

	for range workers {
		waiting.Add(1)
		go func() {
			defer waiting.Done()
			for index := range next {
				work(index)
			}
		}()
	}

	for index := range count {
		select {
		case <-ctx.Done():
			close(next)
			waiting.Wait()
			return ctx.Err()
		case next <- index:
		}
	}
	close(next)
	waiting.Wait()

	return ctx.Err()
}

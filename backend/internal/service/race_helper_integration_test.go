//go:build integration

package service_test

import "sync"

func raceN(n int, fn func() error) []error {
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = fn()
		}(i)
	}
	close(start)
	wg.Wait()
	return errs
}

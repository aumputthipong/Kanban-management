//go:build integration

package service_test

import "sync"

// raceN releases n goroutines at once and returns each call's error, so a test can
// assert on how the losers of a read-then-write race are reported.
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

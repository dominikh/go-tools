package pkg

import "sync"

func fn() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wg.Done()
	}()

	go func() {
		wg.Add(1) // MATCH /should call wg\.Add\(1\) before starting/
		wg.Done()
	}()

	wg.Wait()
}

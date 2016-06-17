package pkg

import "sync"

func fn1() {
	var x sync.Mutex
	x.Lock()
	x.Unlock() // MATCH /empty critical section/
}

func fn2() {
	x := struct {
		m1 struct {
			m2 sync.Mutex
		}
	}{}

	x.m1.m2.Lock()
	x.m1.m2.Unlock() // MATCH /empty critical section/
}

func fn3() {
	var x sync.RWMutex
	x.Lock()
	x.Unlock() // MATCH /empty critical section/

	x.RLock()
	x.RUnlock() // MATCH /empty critical section/

	x.Lock()
	defer x.Unlock()
}

func fn4() {
	x := struct {
		m func() *sync.Mutex
	}{
		m: func() *sync.Mutex {
			return new(sync.Mutex)
		},
	}

	x.m().Lock()
	x.m().Unlock()
}

func fn5() {
	i := 0
	var x sync.Mutex
	x.Lock()
	i++
	x.Unlock()
}

func fn6() {
	x := &sync.Mutex{}
	x.Lock()
	x.Unlock() // MATCH /empty critical section/
}

func fn7() {
	x := &struct {
		sync.Mutex
	}{}

	x.Lock()
	x.Unlock() // MATCH /empty critical section/
}

func fn8() {
	var x sync.Locker
	x = new(sync.Mutex)

	x.Lock()
	x.Unlock() // MATCH /empty critical section/
}

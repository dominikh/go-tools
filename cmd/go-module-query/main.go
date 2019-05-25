package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/renameio"
	"golang.org/x/mod/module"
)

/*
Q: which versions of our module are being used
A: find the latest version of every Go module, find the dependency on our module

Q: what modules have stopped using our module
A: find every module where a version [0..N) uses us, but version N doesn't.
*/

func Fetch(since time.Time) ([]module.Version, time.Time, error) {
	var out []module.Version
	for {
		out2, since2, err := fetch(since, out)
		if err != nil {
			return nil, since, err
		}
		if len(out) == len(out2) {
			break
		}
		out = out2
		since = since2
	}
	return out, since, nil
}

func fetch(since time.Time, out []module.Version) ([]module.Version, time.Time, error) {
	// +1µs because of bug in index.golang.org that returns results
	// >=since instead of >since
	ts := since.Add(1 * time.Microsecond)
	u := `https://index.golang.org/index?since=` + ts.Format(time.RFC3339Nano)
	resp, err := http.Get(u)
	if err != nil {
		return nil, since, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	var entry struct {
		module.Version
		Timestamp time.Time
	}
	for {
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return out, since, err
		}

		out = append(out, entry.Version)
		since = entry.Timestamp
	}

	return out, since, nil
}

func main() {
	cache, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}

	var since time.Time
	b, err := ioutil.ReadFile(filepath.Join(cache, "go-module-query", "last"))
	if err == nil {
		t, err := time.Parse(time.RFC3339Nano, string(b))
		if err != nil {
			log.Fatal(err)
		}
		since = t
		log.Println("Resuming at", since)
	} else if !os.IsNotExist(err) {
		log.Fatal(err)
	}

	out, since, err := Fetch(since)
	if err != nil {
		log.Fatal(err)
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var errs uint64
	for _, v := range out {
		mpath, _ := module.EscapePath(v.Path)
		p := filepath.Join(cache, "go-module-query", mpath, "@v", v.Version+".mod")
		// XXX is this atomic?
		if err := os.MkdirAll(filepath.Join(cache, "go-module-query", mpath, "@v"), 0777); err != nil {
			log.Println(err)
			continue
		}
		if _, err := os.Stat(p); os.IsNotExist(err) {
			fmt.Println("Fetching", v)
			sem <- struct{}{}
			wg.Add(1)
			go func(p string, v module.Version) {
				defer wg.Done()
				defer func() { <-sem }()
				resp, err := http.Get("https://proxy.golang.org/" + path.Join(mpath, "@v", v.Version+".mod"))
				if err != nil {
					atomic.AddUint64(&errs, 1)
					log.Println(err)
					return
				}
				defer resp.Body.Close()
				pf, err := renameio.TempFile("", p)
				if err != nil {
					atomic.AddUint64(&errs, 1)
					log.Println(err)
					return
				}
				defer pf.Cleanup()
				if _, err := io.Copy(pf, resp.Body); err != nil {
					atomic.AddUint64(&errs, 1)
					log.Println(err)
					return
				}
				if err := pf.CloseAtomicallyReplace(); err != nil {
					atomic.AddUint64(&errs, 1)
					log.Println("Couldn't store go.mod:", err)
				}
			}(p, v)
		}
	}

	wg.Wait()

	if errs > 0 {
		log.Println("Couldn't download all go.mod, not storing timestamp")
		return
	}

	if err := renameio.WriteFile(filepath.Join(cache, "go-module-query", "last"), []byte(since.Format(time.RFC3339Nano)), 0666); err != nil {
		log.Println("Couldn't store timestamp:", err)
	}
}

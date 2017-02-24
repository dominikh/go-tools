package functions

import (
	"sync"

	"honnef.co/go/tools/callgraph"
	"honnef.co/go/tools/callgraph/static"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/staticcheck/vrp"
)

var stdlibDescs = map[string]Description{
	"strings.Map":            Description{Pure: true},
	"strings.Repeat":         Description{Pure: true},
	"strings.Replace":        Description{Pure: true},
	"strings.Title":          Description{Pure: true},
	"strings.ToLower":        Description{Pure: true},
	"strings.ToLowerSpecial": Description{Pure: true},
	"strings.ToTitle":        Description{Pure: true},
	"strings.ToTitleSpecial": Description{Pure: true},
	"strings.ToUpper":        Description{Pure: true},
	"strings.ToUpperSpecial": Description{Pure: true},
	"strings.Trim":           Description{Pure: true},
	"strings.TrimFunc":       Description{Pure: true},
	"strings.TrimLeft":       Description{Pure: true},
	"strings.TrimLeftFunc":   Description{Pure: true},
	"strings.TrimPrefix":     Description{Pure: true},
	"strings.TrimRight":      Description{Pure: true},
	"strings.TrimRightFunc":  Description{Pure: true},
	"strings.TrimSpace":      Description{Pure: true},
	"strings.TrimSuffix":     Description{Pure: true},

	"(*net/http.Request).WithContext": Description{Pure: true},
}

type Description struct {
	// The function is known to be pure
	Pure bool
	// The function is known to never return (panics notwithstanding)
	Infinite bool
	// Variable ranges
	Ranges vrp.Ranges
}

type descriptionEntry struct {
	ready  chan struct{}
	result Description
}

type Descriptions struct {
	CallGraph *callgraph.Graph
	mu        sync.Mutex
	cache     map[*ssa.Function]*descriptionEntry
}

func NewDescriptions(prog *ssa.Program) *Descriptions {
	return &Descriptions{
		CallGraph: static.CallGraph(prog),
		cache:     map[*ssa.Function]*descriptionEntry{},
	}
}

func (d *Descriptions) Get(fn *ssa.Function) Description {
	d.mu.Lock()
	fd := d.cache[fn]
	if fd == nil {
		fd = &descriptionEntry{
			ready: make(chan struct{}),
		}
		d.cache[fn] = fd
		d.mu.Unlock()

		{
			fd.result = stdlibDescs[fn.RelString(nil)]
			fd.result.Pure = fd.result.Pure || d.IsPure(fn)
			fd.result.Infinite = fd.result.Infinite || !terminates(fn)
			fd.result.Ranges = vrp.BuildGraph(fn).Solve()
		}

		close(fd.ready)
	} else {
		d.mu.Unlock()
		<-fd.ready
	}
	return fd.result
}

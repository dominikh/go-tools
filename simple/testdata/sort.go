package pkg

import "sort"

type MyIntSlice []int

func (s MyIntSlice) Len() int           { return 0 }
func (s MyIntSlice) Less(i, j int) bool { return true }
func (s MyIntSlice) Swap(i, j int)      {}

func fn() {
	var a []int
	sort.Sort(sort.IntSlice(a)) // MATCH "sort.Ints"

	var b []float64
	sort.Sort(sort.Float64Slice(b)) // MATCH "sort.Float64s"

	var c []string
	sort.Sort(sort.StringSlice(c)) // MATCH "sort.Strings"

	sort.Sort(MyIntSlice(a))

	var d MyIntSlice
	sort.Sort(d)

	var e sort.Interface
	sort.Sort(e)
}

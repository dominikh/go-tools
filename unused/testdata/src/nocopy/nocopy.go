package bar

type myNoCopy1 struct{}
type myNoCopy2 struct{}
type locker struct{}            // MATCH "locker is unused"
type someStruct struct{ x int } // MATCH "someStruct is unused"

func (myNoCopy1) Lock()      {}
func (recv myNoCopy2) Lock() {}
func (locker) Lock()         {}
func (locker) Unlock()       {}
func (someStruct) Lock()     {}

type T struct {
	noCopy1 myNoCopy1
	noCopy2 myNoCopy2
	field1  someStruct // MATCH "field1 is unused"
	field2  locker     // MATCH "field2 is unused"
	field3  int        // MATCH "field3 is unused"
}

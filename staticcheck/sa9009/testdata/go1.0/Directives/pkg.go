package pkg

import _ "unsafe" // for go:linkname

// go:linkname bad1 foo.Bad1 //@ diag(`ineffectual compiler directive`)
func bad1() {
}

//	go:foobar sdlkfjlaksdj //@ diag(`ineffectual compiler directive`)
func bad2() {
}

/* go:foobar 1234*/ //@ diag(`ineffectual compiler directive`)
func bad3() {
}

/*	go:foobar 4321*/ //@ diag(`ineffectual compiler directive`)
func bad4() {
}

 //go:linkname good1 good.One
func good1() {
}

	//go:foobar blahblah
func good2() {
}

 /*go:foobar asdf*/
func good3() {
}

	/*go:foobar asdf*/
func good4() {
}

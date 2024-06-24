package pkg

import _ "unsafe" // go:linkname

// go:linkname bad1 foo.Bad1 //@ diag(`ineffectual compiler directive`)
func bad1() {
}

//	go:foobar sdlkfjlaksdj //@ diag(`ineffectual compiler directive`)
func bad2() {
}

/* go:foobar 1234*/ // cannot be a compiler directive
func bad3() {
}

/*	go:foobar 4321*/ // cannot be a compiler directive
func bad4() {
}

 //go:linkname good1 good.One
func good1() {
}

	//go:foobar blahblah
func good2() {
}

 /*go:foobar asdf*/ // cannot be a compiler directive
func good3() {
}

	/*go:foobar asdf*/ // cannot be a compiler directive
func good4() {
}


// go: probably just talking about Go
func good5() {
}

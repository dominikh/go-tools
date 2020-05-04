// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gcexportdata provides functions for locating, reading, and
// writing export data files containing type information produced by the
// gc compiler.  This package supports go1.7 export data format and all
// later versions.
//
// Although it might seem convenient for this package to live alongside
// go/types in the standard library, this would cause version skew
// problems for developer tools that use it, since they must be able to
// consume the outputs of the gc compiler both before and after a Go
// update such as from Go 1.7 to Go 1.8.  Because this package lives in
// golang.org/x/tools, sites can update their version of this repo some
// time before the Go 1.8 release and rebuild and redeploy their
// developer tools, which will then be able to consume both Go 1.7 and
// Go 1.8 export data files, so they will work before and after the
// Go update. (See discussion at https://golang.org/issue/15651.)
//
package gcexportdata

import (
	"bufio"
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"

	"honnef.co/go/tools/internal/go/gcimporter"
)

type bufferedReader struct {
	r   io.Reader
	buf *bufio.Reader
}

func (r *bufferedReader) Read(b []byte) (int, error) {
	return r.buf.Read(b)
}

func (r *bufferedReader) ReadSlice(delim byte) (line []byte, err error) {
	return r.buf.ReadSlice(delim)
}

// NewReader returns a reader for the export data section of an object
// (.o) or archive (.a) file read from r.  The new reader may provide
// additional trailing data beyond the end of the export data.
func NewReader(r io.Reader) (io.Reader, error) {
	buf := &bufferedReader{
		r:   r,
		buf: bufio.NewReader(r),
	}
	_, err := gcimporter.FindExportData(buf)
	// If we ever switch to a zip-like archive format with the ToC
	// at the end, we can return the correct portion of export data,
	// but for now we must return the entire rest of the file.
	return buf, err
}

// Read reads export data from in, decodes it, and returns type
// information for the package.
// The package name is specified by path.
// File position information is added to fset.
//
// Read may inspect and add to the imports map to ensure that references
// within the export data to other packages are consistent.  The caller
// must ensure that imports[path] does not exist, or exists but is
// incomplete (see types.Package.Complete), and Read inserts the
// resulting package into this map entry.
//
// On return, the state of the reader is undefined.
func Read(in io.Reader, fset *token.FileSet, imports map[string]*types.Package, path string) (*types.Package, error) {
	var data []byte
	if br, ok := in.(*bufferedReader); ok {
		if f, ok := br.r.(*os.File); ok {
			fi, err := f.Stat()
			if err == nil {
				// we expect to be close to the start of the file,
				// which is why we don't bother checking with
				// SEEK_CUR.
				data = make([]byte, fi.Size())
				n, err := io.ReadFull(in, data)
				data = data[:n]
				if err != nil && err != io.ErrUnexpectedEOF {
					data = nil
				}
			}
		}
	}
	if data == nil {
		var err error
		data, err = ioutil.ReadAll(in)
		if err != nil {
			return nil, fmt.Errorf("reading export data for %q: %v", path, err)
		}
	}

	if bytes.HasPrefix(data, []byte("!<arch>")) {
		return nil, fmt.Errorf("can't read export data for %q directly from an archive file (call gcexportdata.NewReader first to extract export data)", path)
	}

	// The App Engine Go runtime v1.6 uses the old export data format.
	// TODO(adonovan): delete once v1.7 has been around for a while.
	if bytes.HasPrefix(data, []byte("package ")) {
		return gcimporter.ImportData(imports, path, path, bytes.NewReader(data))
	}

	// The indexed export format starts with an 'i'.
	if len(data) == 0 || data[0] != 'i' {
		return nil, fmt.Errorf("unknown export data format")
	}
	_, pkg, err := gcimporter.IImportData(fset, imports, data[1:], path)
	return pkg, err
}

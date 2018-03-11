package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

var (
	bodyTagToken = []byte("</body>")
	testScript   = []byte("<script>document.body.insertAdjacentHTML('beforeend', 'injected');</script>\n")
)

func scanAndModifyHTML(sc *bufio.Scanner, b []byte, gzipped bool, cntntLen int) ([]byte, error) {
	buf := newBuffer()

	if _, err := buf.WriteLine(b); err != nil {
		fmt.Println("writing init scan to buffer")
		return nil, err
	}

	sct := len(b) + 1

	for sc.Scan() {
		bs := sc.Bytes()
		sct += len(bs) + 1

		if _, err := buf.WriteLine(bs); err != nil {
			fmt.Println("writing scan to buffer")
			return nil, err
		}

		fmt.Println("here", cntntLen, sct)
		if sct >= cntntLen {
			break
		}
	}

	bfr := newBuffer()
	out := newBuffer()
	var r io.ReadCloser = buf
	var w io.WriteCloser = out

	if gzipped {
		var err error
		r, err = gzip.NewReader(buf)
		if err != nil {
			fmt.Println("creating reader")
			return nil, err
		}

		w = gzip.NewWriter(out)
	}

	if _, err := io.Copy(bfr, r); err != nil {
		fmt.Println("passing/decompressing")
		return nil, err
	}
	if err := r.Close(); err != nil {
		fmt.Println("closing decomp")
		return nil, err
	}
	//buf.Reset()

	bs := bfr.Bytes()
	last := len(bs) - 1
	spl := last + 1

	for i := last; i > 0; i-- {
		if i > len(bs)-len(bodyTagToken) {
			continue
		}

		if bytes.Equal(bs[i:i+len(bodyTagToken)], bodyTagToken) {
			spl = i
			break
		}
	}

	if _, err := w.Write(bs[:spl]); err != nil {
		fmt.Println("writing bytes to spl", spl)
		return nil, err
	}

	if spl <= last {
		if _, err := w.Write(testScript); err != nil {
			fmt.Println("writing test script")
			return nil, err
		}

		if _, err := w.Write(bs[spl:]); err != nil {
			fmt.Println("writing bytes from spl", spl)
			return nil, err
		}
	}

	return out.Bytes(), nil
}

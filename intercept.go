package main

import (
	"bufio"
	"bytes"
	"io"
)

var (
	cntntEncPfx = []byte("Content-Encoding")
	httpPfx     = []byte("HTTP")
	gzipToken   = []byte("gzip")
)

func intercept(dst io.Writer, src io.Reader, secure bool) (written int64, err error) {
	if secure {
		return io.Copy(dst, src)
	}

	sc := bufio.NewScanner(src)
	var gzip, body bool

	for sc.Scan() {
		bs := sc.Bytes()

		if isHTTP(bs) {
			body = false
			gzip = false
		}

		if isGzipped(bs) {
			gzip = true
		}

		if body {
			bs = modifiedHTML(sc, gzip, bs)
		}

		if len(bs) == 0 {
			body = !body
		}

		n, err := write(dst, bs)
		written += n
		if err != nil {
			return written, err
		}
	}

	if err := sc.Err(); err != nil && err != io.EOF {
		return written, err
	}

	return written, nil
}

func write(w io.Writer, b []byte) (int64, error) {
	nb, err := w.Write(b)
	if err != nil {
		return int64(nb), err
	}

	nn, err := w.Write([]byte("\n"))

	return int64(nb) + int64(nn), err
}

func isHTTP(b []byte) bool {
	return bytes.HasPrefix(b, httpPfx)
}

func isCntntEnc(b []byte) bool {
	return bytes.HasPrefix(b, cntntEncPfx)
}

func isGzip(b []byte) bool {
	return len(b) > len(cntntEncPfx) && bytes.Contains(b[len(cntntEncPfx):], gzipToken)
}

func isGzipped(b []byte) bool {
	return isCntntEnc(b) && isGzip(b)
}

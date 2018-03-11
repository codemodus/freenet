package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode"
)

var (
	cntntEncPfx = []byte("Content-Encoding:")
	cntntTypPfx = []byte("Content-Type:")
	cntntLenPfx = []byte("Content-Length:")
	httpPfx     = []byte("HTTP")
	gzipToken   = []byte("gzip")
	htmlToken   = []byte("text/html")
)

func intercept(dst io.Writer, src io.Reader, secure bool) (written int64, err error) {
	if secure {
		return io.Copy(dst, src)
	}

	return scanAndModify(dst, src)
}

func scanAndModify(dst io.Writer, src io.Reader) (written int64, err error) {
	sc := bufio.NewScanner(src)
	var body, html, gzip, brk, clPrinted bool
	var clen int

	for sc.Scan() {
		bs := sc.Bytes()

		if isCntntLen(bs) {
			clen = cntntLen(bs)
			continue
		}

		if isPad(bs) {
			if !body && clen == 0 {
				n, err := writeLine(dst, nil)
				written += n
				if err != nil {
					return written, err
				}
				break
			}

			body = !body
			continue
		}

		html = triggerHTML(bs, html)
		gzip = triggerGzipped(bs, gzip)

		if isHTTP(bs) {
			if body {
				n, err := writeLine(dst, nil)
				written += n
				if err != nil {
					return written, err
				}
			}

			body = false
			html = false
			gzip = false
			clen = 0
			brk = false
			clPrinted = false
		}

		if body {
			if html {
				var err error

				bs, err = scanAndModifyHTML(sc, bs, gzip, clen)
				if err != nil {
					return written, err
				}

				clen = len(bs)

				brk = true
			}

			if !clPrinted {
				clPrinted = true
				tl := strconv.Itoa(clen)
				x := "Content-Length: " + tl + "\n"
				n, err := writeLine(dst, []byte(x))
				written += n
				if err != nil {
					return written, err
				}
			}
		}

		n, err := writeLine(dst, bs)
		written += n
		if err != nil {
			return written, err
		}

		if brk {
			break
		}
	}

	return written, sc.Err()
}

func writeLine(w io.Writer, b []byte) (int64, error) {
	n := 0

	fmt.Print(string(b))
	if len(b) > 0 {
		nb, err := w.Write(b)
		n = nb
		if err != nil {
			return int64(n), err
		}
	}

	nn, err := w.Write([]byte("\n"))
	fmt.Println()

	return int64(n + nn), err
}

func isHTTP(b []byte) bool {
	return bytes.HasPrefix(b, httpPfx)
}

func cntntLen(b []byte) int {
	for i := len(b) - 1; i >= 0; i-- {
		if !unicode.IsDigit(rune(b[i])) {
			cl, _ := strconv.Atoi(string(b[i+1:]))
			return cl
		}
	}

	return 0
}

func isCntntLen(b []byte) bool {
	return bytes.HasPrefix(b, cntntLenPfx)
}

func triggerHTML(b []byte, v bool) bool {
	if isHTML(b) {
		return true
	}

	return v
}

func isHTML(b []byte) bool {
	return bytes.HasPrefix(b, cntntTypPfx) && bytes.Contains(b[len(cntntTypPfx):], htmlToken)
}

func triggerGzipped(b []byte, v bool) bool {
	if isGzipped(b) {
		return true
	}

	return v
}

func isGzipped(b []byte) bool {
	return bytes.HasPrefix(b, cntntEncPfx) && bytes.Contains(b[len(cntntEncPfx):], gzipToken)
}

func isPad(b []byte) bool {
	return len(b) == 0
}

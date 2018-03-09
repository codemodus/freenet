package main

import (
	"bufio"
	"fmt"
)

func modifiedHTML(sc *bufio.Scanner, gzip bool, b []byte) []byte {
	fmt.Println(gzip, string(b))
	return b
}

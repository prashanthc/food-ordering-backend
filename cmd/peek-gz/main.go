package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode"
)

const url = "https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase1.gz"

func main() {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("download failed: %v", err)
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatalf("gzip open failed: %v", err)
	}
	defer gz.Close()

	br := bufio.NewReaderSize(gz, 64*1024)

	var sb strings.Builder
	found := 0

	for found < 5 {
		b, err := br.ReadByte()
		if err != nil {
			break
		}
		r := rune(b)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToUpper(r))
			continue
		}
		if word := sb.String(); len(word) >= 8 && len(word) <= 10 {
			found++
			fmt.Printf("%d: %s\n", found, word)
		}
		sb.Reset()
	}

	if found == 0 {
		fmt.Println("no tokens found — check the file format")
	}
}

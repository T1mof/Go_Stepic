package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	androidRe = regexp.MustCompile("Android")
	msieRe    = regexp.MustCompile("MSIE")
	emailRe   = regexp.MustCompile("@")
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(out, "error opening file: %v\n", err)
		return
	}
	defer file.Close()

	seenBrowsers := make(map[string]bool)
	uniqueBrowsers := 0
	var foundUsers strings.Builder

	scanner := bufio.NewScanner(file)
	index := 0

	for scanner.Scan() {
		var user map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &user); err != nil {
			fmt.Fprintf(out, "error decoding JSON: %v\n", err)
			continue
		}

		isAndroid := false
		isMSIE := false

		browsers, ok := user["browsers"].([]interface{})
		if !ok {
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				continue
			}

			if androidRe.MatchString(browser) {
				isAndroid = true
				if !seenBrowsers[browser] {
					seenBrowsers[browser] = true
					uniqueBrowsers++
				}
			}

			if msieRe.MatchString(browser) {
				isMSIE = true
				if !seenBrowsers[browser] {
					seenBrowsers[browser] = true
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			index++
			continue
		}

		email, _ := user["email"].(string)
		foundUsers.WriteString(fmt.Sprintf("[%d] %s <%s>\n",
			index,
			user["name"],
			emailRe.ReplaceAllString(email, " [at] "),
		))
		index++
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(out, "error reading file: %v\n", err)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", uniqueBrowsers)
}

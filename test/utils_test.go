package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go test -timeout 30s github.com/slicken/hc -run ^(TestData)$

func Test(t *testing.T) {
	fmt.Println(time.Now().Unix())
}

func TestReadDir(t *testing.T) {

	files, err := ioutil.ReadDir("data")
	if err != nil {
		t.Fatal(err)
	}

	// initialise the map
	// m := make(map[string]string)

	// read filenames from directory
	for _, file := range files {
		// file is directory
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fn := file.Name()
		s := strings.SplitAfter(fn, "-")

		fmt.Printf("Symbol: %v\t Timeframe: %v", s[0], s[1])
	}

}

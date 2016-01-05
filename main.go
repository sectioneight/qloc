package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var (
	procs = flag.Int("p", runtime.NumCPU(), "how many processes to use for counting lines")
	exts  = flag.String("ext", "", "only extensions ('go,html,js')")
)

func ShouldExamine(ext string) bool {
	if *exts == "" {
		return true
	}
	ext = strings.ToLower("," + strings.TrimPrefix(ext, ".") + ",")
	return strings.Contains(*exts, ext)
}

func main() {
	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		path = "."
	}

	if *exts != "" {
		*exts = "," + *exts + ","
	}

	work := make(chan string, 100)
	results := make(chan CountByExt, *procs)
	go IterateDir(path, work)

	for i := 0; i < *procs; i++ {
		go FileWorker(work, results)
	}

	total := make(CountByExt)
	for N := *procs; N > 0; {
		select {
		case result := <-results:
			for _, s := range result {
				total.Add(s)
			}
			N--
		}
	}
	fmt.Println()

	counts := make(Counts, 0, len(total))
	for _, c := range total {
		counts = append(counts, c)
	}
	sort.Sort(ByCode{counts})

	// fmt.Printf("%-12s %12s %12s %12s %12s\n", "extension", "files", "binary", "blank", "code")
	fmt.Printf("path")
	for _, count := range counts {
		fmt.Printf(", %s files, %s code, %s blank", count.Ext, count.Ext, count.Ext)
	}
	fmt.Printf("\n")

	fmt.Printf(path)
	for _, count := range counts {
		fmt.Printf(", %12d, %12d, %12d", count.Files, count.Code, count.Blank)
	}
	fmt.Printf("\n")
}

func FileWorker(files chan string, result chan CountByExt) {
	total := make(CountByExt)
	defer func() { result <- total }()

	for file := range files {
		count, err := CountLines(file)
		if err != nil {
			continue
		}
		total.Add(count)
	}
}

func IterateDir(root string, work chan string) {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			info.Mode()
			filename := info.Name()
			if len(filename) > 1 && filename[0] == '.' {
				return filepath.SkipDir
			}
			return nil
		}

		// if a filename contains ~ we assume it's a temporary file
		if strings.Contains(filepath.Base(path), "~") {
			return nil
		}

		if !ShouldExamine(filepath.Ext(path)) {
			return nil
		}

		work <- path
		return nil
	}

	err := filepath.Walk(root, walk)
	if err != nil {
		log.Println(err)
	}

	close(work)
}

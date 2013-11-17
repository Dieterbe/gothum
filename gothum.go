package main

import (
	"flag"
	"fmt"
	"github.com/Dieterbe/gothum/workers"
	"log"
	"os"
	"runtime"
	"sync"
)

var resize_threads int
var in, out string

func init() {
	flag.IntVar(&resize_threads, "resize_threads", 2, "how many concurrent resizer threads (each can max out a cpu core)")
	flag.StringVar(&in, "in", "", "input directory")
	flag.StringVar(&out, "out", "", "output (thumbnail) directory")
}

func dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	if in == "" {
		log.Fatal("no input directory specified")
	}
	if out == "" {
		log.Fatal("no output directory specified")
	}
	var wg sync.WaitGroup
	paths := workers.IoWorker(in)
	var i int
	runtime.GOMAXPROCS(resize_threads)
	wg.Add(resize_threads)
	for i = 1; i <= resize_threads; i++ {
		go workers.ResizeWorker(i, paths, out, &wg)
	}
	wg.Wait()
}

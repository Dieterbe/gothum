package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func IoWorker(dir_in string) (paths chan string) {
	paths = make(chan string, 1000)
	go func() {
		list, err := ioutil.ReadDir(dir_in)
		dieIfError(err)
		abspath, err := filepath.Abs(dir_in)
		dieIfError(err)
		for _, f := range list {
			name := f.Name()
			ext := filepath.Ext(name)
			mime := mime.TypeByExtension(ext)
			if !strings.HasPrefix(mime, "image/") {
				continue
			}
			paths <- fmt.Sprintf("%s/%s", abspath, name)
		}
		close(paths)
	}()
	return paths
}
func ResizeWorker(i int, paths chan string, out string, wg *sync.WaitGroup) {
	for abspath_in := range paths {
		fmt.Println("## [", i, "]", abspath_in)
		fileUri_in := fmt.Sprintf("file://%s", abspath_in)
		h := md5.New()
		io.WriteString(h, fileUri_in)
		pathmd5 := fmt.Sprintf("%x", h.Sum(nil))
		path_out := fmt.Sprintf("%s/%s.png", out, pathmd5)
		_, err := os.Stat(path_out)
		if err == nil {
			fmt.Printf("[%d] %s already exists! skipping\n", i, path_out)
			continue
		}
		file_in, err := os.Open(abspath_in)
		dieIfError(err)
		config, _, err := image.DecodeConfig(file_in)
		if err != nil {
			fmt.Printf("WARNING. Could not decode image config '%s', skipping: %s\n", abspath_in, err)
			continue
		}
		file_in.Seek(0, os.SEEK_SET)
		var img_in image.Image
		img_in, _, err = image.Decode(file_in)
		if err != nil {
			fmt.Printf("WARNING. Could not decode image '%s', skipping: %s\n", abspath_in, err)
			continue
		}

		file_in.Close()
		width := 0
		height := 0
		if config.Width > config.Height {
			width = 256
		} else {
			height = 256
		}
		img_out := imaging.Resize(img_in, width, height, imaging.CatmullRom)
		fmt.Printf("[%d] --> %s\n", i, path_out)
		file_out, err := os.Create(path_out)
		if err != nil {
			log.Fatal(err)
		}
		err = png.Encode(file_out, img_out)
		file_out.Close()
		dieIfError(err)
	}
	wg.Done()
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
	paths := IoWorker(in)
	var i int
	runtime.GOMAXPROCS(resize_threads)
	wg.Add(resize_threads)
	for i = 1; i <= resize_threads; i++ {
		go ResizeWorker(i, paths, out, &wg)
	}
	wg.Wait()
}

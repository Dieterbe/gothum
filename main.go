package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"github.com/nfnt/resize"
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
	"strings"
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
	list, err := ioutil.ReadDir(in)
	dieIfError(err)
	abspath, err := filepath.Abs(in)
	dieIfError(err)
	for _, f := range list {
		name := f.Name()
		ext := filepath.Ext(name)
		mime := mime.TypeByExtension(ext)
		if !strings.HasPrefix(mime, "image/") {
			continue
		}
		abspath_in := fmt.Sprintf("%s/%s", abspath, name)
		fileUri_in := fmt.Sprintf("file://%s", abspath_in)
		fmt.Println("##", abspath_in)
		h := md5.New()
		io.WriteString(h, fileUri_in)
		pathmd5 := fmt.Sprintf("%x", h.Sum(nil))
		file_in, err := os.Open(abspath_in)
		dieIfError(err)
		config, _, err := image.DecodeConfig(file_in)
		file_in.Seek(0, 0)
		var img_in image.Image
		img_in, _, err = image.Decode(file_in)
		if err != nil {
			fmt.Printf("WARNING: unsupported image format: '%s'. skipping.\n", mime)
			continue
		}

		file_in.Close()
		if err != nil {
			fmt.Printf("WARNING. Could not decode '%s', so i'm skipping it: %s\n", name, err)
			continue
		}
		width := uint(0)
		height := uint(0)
		if config.Width > config.Height {
			width = 256
		} else {
			height = 256
		}
		img_out := resize.Resize(width, height, img_in, resize.NearestNeighbor)
		path_out := fmt.Sprintf("%s/%s.png", out, pathmd5)
		fmt.Printf("--> %s\n", path_out)
		file_out, err := os.Create(path_out)
		if err != nil {
			log.Fatal(err)
		}
		defer file_out.Close()
		err = png.Encode(file_out, img_out)
		dieIfError(err)
	}
}

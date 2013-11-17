package workers

import (
	"crypto/md5"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

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
		err := Resize(fmt.Sprintf("#%d", i), abspath_in, out)
		dieIfError(err)
	}
	wg.Done()
}

func Resize(log_prefix string, abspath_in string, out string) (err error) {
	fmt.Printf("%s in:  %s\n", log_prefix, abspath_in)
	fileUri_in := fmt.Sprintf("file://%s", abspath_in)
	h := md5.New()
	io.WriteString(h, fileUri_in)
	pathmd5 := fmt.Sprintf("%x", h.Sum(nil))
	path_out := fmt.Sprintf("%s/%s.png", out, pathmd5)
	fmt.Printf("%s out: %s\n", log_prefix, path_out)
	_, err = os.Stat(path_out)
	if err == nil {
		fmt.Printf("%s thumb already exists! skipping\n", log_prefix)
		return nil
	}
	err = exec.Command("gm", "convert", "-size", "256x256", abspath_in, "-auto-orient", "-resize", "256x256", path_out).Run()
	if err == nil {
		fmt.Printf("%s thumb done by graphicsmagick\n", log_prefix)
		return nil
	}
	fmt.Printf("%s graphicsmagick returned error. trying in pure go. error: %s\n", log_prefix, err.Error())
	file_in, err := os.Open(abspath_in)
	dieIfError(err)
	config, _, err := image.DecodeConfig(file_in)
	if err != nil {
		fmt.Printf("%s WARNING. Could not decode image config '%s', skipping: %s\n", log_prefix, abspath_in, err)
		return nil
	}
	file_in.Seek(0, os.SEEK_SET)
	var img_in image.Image
	img_in, _, err = image.Decode(file_in)
	if err != nil {
		fmt.Printf("%s WARNING. Could not decode image '%s', skipping: %s\n", log_prefix, abspath_in, err)
		return nil
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
	file_out, err := os.Create(path_out)
	if err != nil {
		return err
	}
	fmt.Printf("%s thumb done in pure Go\n", log_prefix)
	err = png.Encode(file_out, img_out)
	file_out.Close()
	return nil
}

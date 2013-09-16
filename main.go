package main

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	nfnt_resize "github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"os/exec"
	moustaschio_resize "resize"
	"strings"
	"time"
)

func scanDir(path string) (files []string, err error) {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}
	for _, r := range entries {
		n := strings.ToUpper(r.Name())
		if strings.HasSuffix(n, ".JPG") || strings.HasSuffix(n, ".JPEG") {
			files = append(files, path+"/"+r.Name())
		}
	}
	return
}

func timeTrack(start time.Time, name string, n int) string {
	elapsed := time.Since(start)
	avg := time.Duration(int64(elapsed) / int64(n))
	s := fmt.Sprintf("%s took %s, file average %s\n", name, elapsed, avg)
	fmt.Println(s)
	return s
}

func resizeNfnt(origName, newName string, interp nfnt_resize.InterpolationFunction) (int, int64) {
	origFile, _ := os.Open(origName)
	origImage, _ := jpeg.Decode(origFile)
	origFileStat, _ := origFile.Stat()
	origFile.Close()

	var resized image.Image
	p := origImage.Bounds().Size()
	if p.X > p.Y {
		resized = nfnt_resize.Resize(150, 0, origImage, interp)
	} else {
		resized = nfnt_resize.Resize(0, 150, origImage, interp)
	}
	b := new(bytes.Buffer)
	jpeg.Encode(b, resized, nil)
	blen := b.Len()
	cacheFile, err := os.Create(newName)
	defer cacheFile.Close()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	b.WriteTo(cacheFile)

	return blen, origFileStat.Size()
}

func resizeNfntNearestNeighbor(origName, newName string) (int, int64) {
	return resizeNfnt(origName, newName, nfnt_resize.NearestNeighbor)
}

func moustachioResample(origName, newName string) (int, int64) {
	return resizeMoustachio(origName, newName, moustaschio_resize.Resample)
}

func moustachioResize(origName, newName string) (int, int64) {
	return resizeMoustachio(origName, newName, moustaschio_resize.Resize)
}

func resizeMoustachio(origName, newName string, method func(image.Image, image.Rectangle, int, int) image.Image) (int, int64) {
	origFile, _ := os.Open(origName)
	origImage, _ := jpeg.Decode(origFile)
	origFileStat, _ := origFile.Stat()
	origFile.Close()

	var resized image.Image
	p := origImage.Bounds().Size()
	if p.X > p.Y {
		resized = method(origImage, origImage.Bounds(), 150, 100)
	} else {
		resized = method(origImage, origImage.Bounds(), 100, 150)
	}
	b := new(bytes.Buffer)
	jpeg.Encode(b, resized, nil)
	blen := b.Len()
	cacheFile, err := os.Create(newName)
	defer cacheFile.Close()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	b.WriteTo(cacheFile)

	return blen, origFileStat.Size()
}

func resizeImaging(origName, newName string) (int, int64) {
	origFileStat, _ := os.Stat(origName)
	origImage, _ := imaging.Open(origName)
	var resized image.Image
	p := origImage.Bounds().Size()
	if p.X > p.Y {
		resized = imaging.Resize(origImage, 150, 100, imaging.Box)
	} else {
		resized = imaging.Resize(origImage, 100, 150, imaging.Box)
	}
	b := new(bytes.Buffer)
	jpeg.Encode(b, resized, nil)
	blen := b.Len()
	cacheFile, err := os.Create(newName)
	defer cacheFile.Close()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	b.WriteTo(cacheFile)
	return blen, origFileStat.Size()

}

func imageMagickThumbnail(origName, newName string) (int, int64) {
	origFileStat, _ := os.Stat(origName)
	var args = []string{
		"-define", "jpeg:size=300x300",
		"-thumbnail", "150x150>",
		origName, newName,
	}
	cmd := exec.Command("/usr/bin/convert", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	newFileStat, _ := os.Stat(newName)
	return int(newFileStat.Size()), origFileStat.Size()
}

func imageMagickResize(origName, newName string) (int, int64) {
	origFileStat, _ := os.Stat(origName)
	var args = []string{
		"-resize", "150x150>",
		origName, newName,
	}
	cmd := exec.Command("/usr/bin/convert", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	newFileStat, _ := os.Stat(newName)
	return int(newFileStat.Size()), origFileStat.Size()
}

func graphicsMagickThumbnail(origName, newName string) (int, int64) {
	origFileStat, _ := os.Stat(origName)
	var args = []string{
		"convert",
		"-define", "jpeg:size=300x300",
		"-thumbnail", "150x150>",
		origName, newName,
	}
	cmd := exec.Command("/usr/bin/gm", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return 0, origFileStat.Size()
	}
	newFileStat, _ := os.Stat(newName)
	return int(newFileStat.Size()), origFileStat.Size()
}

func resize(files []string, desc string, m func(string, string) (int, int64)) string {
	start := time.Now()

	for i, origPath := range files {
		newPath := fmt.Sprintf("%s.thumb.%s.jpg", origPath, desc)
		if printSingleFile {
			fmt.Printf("File %d: ", i)
		}
		imgStart := time.Now()
		n, o := m(origPath, newPath)
		ratio := float64(n) / float64(o) * 100.0
		dur := time.Since(imgStart)
		if printSingleFile {
			fmt.Printf("re-encoded to size=%d (%.1f%%) in %s\n", n, ratio, dur)
		}
	}
	return timeTrack(start, desc, len(files))
}

var printSingleFile bool

func main() {
	dir := "."
	if len(os.Args) > 1 {
		fmt.Println(os.Args)
		dir = os.Args[1]
	}
	printSingleFile = true
	files, _ := scanDir(dir)
	if len(files) == 0 {
		fmt.Println("no jpg files found in", dir)
		return
	}
	if len(files) > 3 {
		//files = files[0:3]
	}
	var results []string
	if _, err := os.Stat("/usr/bin/gm"); err == nil {
		results = append(results, resize(files, "GraphicsMagick_thumbnail", graphicsMagickThumbnail))
	}
	if _, err := os.Stat("/usr/bin/convert"); err == nil {
		results = append(results, resize(files, "ImageMagick_thumbnail", imageMagickThumbnail))
		results = append(results, resize(files, "ImageMagick_resize", imageMagickResize))
	}
	results = append(results, resize(files, "imaging_Box", resizeImaging))
	results = append(results, resize(files, "moustaschio_resize", moustachioResize))
	results = append(results, resize(files, "Nfnt_NearestNeighbor", resizeNfntNearestNeighbor))

	for _, s := range results {
		fmt.Println(s)
	}
}

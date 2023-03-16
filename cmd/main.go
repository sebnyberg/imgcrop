package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"os"

	"github.com/sebnyberg/imgcut"
	"golang.org/x/image/bmp"
)

func main() {
	inflags := os.O_RDONLY
	f, err := os.OpenFile("testdata/img.bmp", inflags, 0)
	if err != nil {
		log.Fatalln(err)
	}
	cfg, err := bmp.DecodeConfig(f)
	if err != nil {
		log.Fatalln(err)
	}
	width := cfg.Width
	height := cfg.Height
	dx := 16000
	dy := 1000
	for i := 0; ; i++ {
		f.Seek(0, 0) // reset
		bb := imgcut.NewBMP(f)
		offx := rand.Intn(width - dx)
		offy := rand.Intn(height - dy)
		rect := image.Rect(offx, offy, offx+dx, offy+dy)
		err = bb.Cut(rect, io.Discard)
		if err != nil {
			log.Fatalln(err)
		}
		if i%100 == 0 {
			fmt.Println(i)
		}
	}
}

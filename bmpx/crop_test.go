package bmpx

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testing"

	seekable "github.com/SaveTheRbtz/zstd-seekable-format-go"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

const (
	tiffBigPath             = "./testdata/big.tif"
	tiffBigUncompressedPath = "./testdata/big-uncompressed.tif"
	tiffBigURL              = "https://esahubble.org/media/archives/images/original/heic0707a.tif"
	tiffSamplePath          = "./testdata/sample.tif"
	tiffSampleURL           = "https://github.com/libvips/libvips/raw/master/test/test-suite/images/sample.tif"
	bmpBigPath              = "./testdata/big.bmp"
)

const inflags = os.O_RDONLY
const outflags = os.O_WRONLY | os.O_TRUNC | os.O_CREATE

func TestCrop(t *testing.T) {
	// todo(sn): need better, more controlled tests
	inflags := os.O_RDONLY
	f, err := os.OpenFile("testdata/big.bmp", inflags, 0)
	require.NoError(t, err)
	defer f.Close()
	outflags := os.O_RDWR | os.O_TRUNC | os.O_CREATE
	outf, err := os.OpenFile("testdata/big-cropped.bmp", outflags, 0640)
	require.NoError(t, err)
	defer outf.Close()
	rect := image.Rect(24500, 10000, 24900, 11000)
	err = Crop(f, outf, rect)
	require.NoError(t, err)
	_, err = outf.Seek(0, 0)
	require.NoError(t, err)
	img, err := bmp.Decode(outf)
	require.NoError(t, err)
	require.Equal(t, img.Bounds().Dx(), rect.Dx())
	require.Equal(t, img.Bounds().Dy(), rect.Dy())
}

func TestResize(t *testing.T) {
	for i := 0; i < 100; i++ {
		// Create random rgba
		w := 100 + rand.Intn(200)
		h := 100 + rand.Intn(200)
		img := randRGBA(w, h)
		var orig, cropped bytes.Buffer
		err := bmp.Encode(&orig, img)
		require.NoError(t, err)
		offx := rand.Intn(w)
		offy := rand.Intn(h)
		dx := rand.Intn(w - offx)
		dy := rand.Intn(h - offy)
		region := image.Rect(offx, offy, offx+dx, offy+dy)
		err = Crop(&orig, &cropped, region)
		require.NoError(t, err)
		got, err := bmp.Decode(&cropped)
		require.NoError(t, err)
		want, err := stdlibCrop(img, region)
		require.NoError(t, err)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				gotxy := got.At(x, y)
				wantxy := want.At(x, y)
				require.Equal(t, gotxy, wantxy, "(%v,%v)", x, y)
			}
		}
	}
}

func BenchmarkCrop(b *testing.B) {
	inflags := os.O_RDONLY
	f, err := os.OpenFile(bmpBigPath, inflags, 0)
	require.NoError(b, err)
	cfg, err := bmp.DecodeConfig(f)
	require.NoError(b, err)
	width := cfg.Width
	height := cfg.Height

	for dx := 100; dx <= width; dx *= 8 {
		for dy := 100; dy <= height; dy *= 8 {
			b.Run(fmt.Sprintf("%dX%d", dx, dy), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					outf, err := os.OpenFile("testdata/big-cropped.bmp", outflags, 0640)
					require.NoError(b, err)
					f.Seek(0, io.SeekStart)
					offx := rand.Intn(width - dx)
					offy := rand.Intn(height - dy)
					rect := image.Rect(offx, offy, offx+dx, offy+dy)
					err = Crop(f, outf, rect)
					require.NoError(b, err)
				}
			})
		}
	}
}

func bmpToZSTDFile(from, to string) error {
	f, err := os.OpenFile(from, inflags, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	outf, err := os.OpenFile(to, outflags, 0644)
	if err != nil {
		return err
	}
	defer outf.Close()
	enc, err := zstd.NewWriter(outf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return err
	}
	defer enc.Close()
	_, err = io.Copy(enc, f)
	return err
}

func bmpToSeekableZSTDFile(from, to string) error {
	f, err := os.OpenFile(from, inflags, 0)
	defer f.Close()
	if err != nil {
		return err
	}
	outf, err := os.OpenFile(to, outflags, 0644)
	defer outf.Close()
	if err != nil {
		return err
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer enc.Close()
	w, err := seekable.NewWriter(outf, enc)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, f)
	return err
}

func init() {
	doInit()
}

func doInit() {
	os.Mkdir("testdata", 0750)
	if err := dl(tiffBigURL, tiffBigPath); err != nil {
		log.Fatalf("dl image at %q to %q err, %v", tiffBigURL, tiffBigPath, err)
	}
	in, err := os.OpenFile(tiffBigPath, inflags, 0)
	if err != nil {
		log.Fatalf("open big tiff at %q err %v", tiffBigPath, err)
	}
	defer in.Close()

	// encode big tiff to various formats
	type encodeTask struct {
		path   string
		encode func(out io.Writer, img image.Image, any any) error
	}
	tiffEncode := func(out io.Writer, img image.Image, any any) error {
		return tiff.Encode(out, img, nil)
	}
	_ = tiffEncode
	bmpEncode := func(out io.Writer, img image.Image, any any) error {
		return bmp.Encode(out, img)
	}
	_ = bmpEncode

	var img image.Image

	for _, e := range []encodeTask{
		{
			path:   tiffBigUncompressedPath,
			encode: tiffEncode,
		},
		{
			path:   bmpBigPath,
			encode: bmpEncode,
		},
	} {
		_, err := os.Stat(e.path)
		if err == nil {
			continue // target already exists
		}
		if img == nil {
			img, err = tiff.Decode(in)
			if err != nil {
				log.Fatalf("decode big tiff at %q err, %v", e.path, err)
			}
		}
		out, err := os.OpenFile(e.path, outflags, 0644)
		if err != nil {
			log.Fatalf("open img output path %q err, %v", e.path, err)
		}
		err = e.encode(out, img, nil)
		if err != nil {
			log.Fatalf("open img output path %q err, %v", e.path, err)
		}
		out.Close()
	}
}

func dl(url, path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	f, err := os.OpenFile(path, outflags, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func stdlibCrop(img image.Image, rect image.Rectangle) (image.Image, error) {
	subimg, ok := img.(subImager)
	if !ok {
		return nil, errors.New("image does not support sub-imaging")
	}
	img = subimg.SubImage(rect)
	fixedCoords := draw.Draw(img, rect, img)
	return subimg.SubImage(rect), nil
}

func randRGBA(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	row := make([]byte, img.Stride)
	for i := 0; i < h; i++ {
		_, err := rand.Read(row)
		if err != nil {
			log.Fatalln(err)
		}
		for j := 0; j < w; j++ {
			r := uint8(row[j*4])
			g := uint8(row[j*4+1])
			b := uint8(row[j*4+2])
			a := uint8(row[j*4+3])
			img.SetNRGBA(j, i, color.NRGBA{r, g, b, a})
		}
	}
	return img
}

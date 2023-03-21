package imgcrop_test

import (
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testing"

	govips "github.com/davidbyttow/govips/v2/vips"
	"github.com/sebnyberg/imgcrop/vipsx"
	"github.com/stretchr/testify/require"
	vips "github.com/vipsimage/vips"
	"golang.org/x/image/tiff"
)

const (
	tiffBigPath    = "./testdata/big.tif"
	tiffBigURL     = "https://esahubble.org/media/archives/images/original/heic0707a.tif"
	tiffSamplePath = "./testdata/sample.tif"
	tiffSampleURL  = "https://github.com/libvips/libvips/raw/master/test/test-suite/images/sample.tif"
)

const (
	inflags  = os.O_RDONLY
	outflags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
)

func vips2crop(name string, area image.Rectangle, out string) error {
	img, err := vips.NewFromFile(name)
	if err != nil {
		return err
	}
	err = img.Crop(area.Min.X, area.Min.Y, area.Dx(), area.Dy())
	if err != nil {
		return err
	}
	err = img.TIFFSave(out)
	return err
}

func BenchmarkTIFF(b *testing.B) {
	// Todo(sn): move this to package with finalizer?
	govips.LoggingSettings(nil, govips.LogLevelCritical)
	govips.Startup(nil)
	defer govips.Shutdown()

	type cropFn func(r io.Reader, area image.Rectangle, out io.Writer) error
	type cropBench struct {
		name string
		fn   cropFn
	}

	dims := func(fname string, b *testing.B) (int, int) {
		f, err := os.OpenFile(fname, inflags, 0)
		defer f.Close()
		require.NoError(b, err)
		cfg, err := tiff.DecodeConfig(f)
		require.NoError(b, err)
		return cfg.Width, cfg.Height
	}

	for _, imgPath := range []string{
		tiffBigPath,
		// tiffSamplePath,
	} {
		width, height := dims(imgPath, b)

		b.Run(imgPath, func(b *testing.B) {
			for dx := 8; dx <= width; dx *= 8 {
				for dy := 8; dy <= height; dy *= 8 {
					for _, cropper := range []cropBench{
						{
							name: "govips",
							fn:   vipsx.Crop,
						},
						{
							name: "vipsimage",
							fn:   nil,
						},
					} {
						b.Run(fmt.Sprintf("%dX%d-%s", dx, dy, cropper.name), func(b *testing.B) {
							var f *os.File
							var err error
							if cropper.name != "vipsimage" {
								f, err = os.OpenFile(tiffBigPath, inflags, 0)
								defer f.Close()
								require.NoError(b, err)
							}
							for i := 0; i < b.N; i++ {
								offx := rand.Intn(width - dx)
								offy := rand.Intn(height - dy)
								rect := image.Rect(offx, offy, offx+dx, offy+dy)
								if cropper.name == "vipsimage" {
									vips2crop(imgPath, rect, "tmp.tif")
								} else {
									err = cropper.fn(f, rect, io.Discard)
									require.NoError(b, err)
									f.Seek(0, io.SeekStart)
								}
							}
						})
					}
				}
			}
		})
	}
}

func init() {
	doInit()
}

func doInit() {
	for _, x := range []struct {
		url  string
		path string
	}{
		{tiffBigURL, tiffBigPath},
		{tiffSampleURL, tiffSamplePath},
	} {
		err := dl(x.url, x.path)
		if err != nil {
			log.Fatalf("dl image at %q to %q err, %v", x.url, x.path, err)
		}
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

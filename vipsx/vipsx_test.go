package vipsx

import (
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const tiffBigPath = "./testdata/big.tif"
const tiffBigURL = "https://esahubble.org/media/archives/images/original/heic0707a.tif"
const tiffSamplePath = "./testdata/sample.tif"
const tiffSampleURL = "https://github.com/libvips/libvips/blob/master/test/test-suite/images/sample.tif"
const inflags = os.O_RDONLY
const outflags = os.O_RDWR | os.O_CREATE | os.O_TRUNC

func TestVips(t *testing.T) {
	in, err := os.OpenFile("./testdata/sample.tif", inflags, 0)
	require.NoError(t, err)
	// defer in.Close()
	region := image.Rect(0, 0, 10, 20)
	out, err := os.OpenFile("./testdata/sample-cropped.tif", outflags, 0644)
	require.NoError(t, err)
	// defer out.Close()
	require.NoError(t, Crop(in, region, out), "crop")
}

func init() {
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

package imgcrop

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff"
)

func convToStdlibTiff(a, b string, t *testing.T) {
	f, err := os.OpenFile(a, inflags, 0)
	defer f.Close()
	require.NoError(t, err)
	img, err := tiff.Decode(f)
	require.NoError(t, err)
	out, err := os.OpenFile(b, outflags, 0644)
	defer out.Close()
	require.NoError(t, err)
	err = tiff.Encode(out, img, nil)
	require.NoError(t, err)
}

func TestA(t *testing.T) {
	convToStdlibTiff("testdata/img.tiff", "testdata/uncompressed.tiff", t)
	f, err := os.OpenFile("testdata/uncompressed.tiff", inflags, 0)
	defer f.Close()
	require.NoError(t, err)
	a, err := tiffDecodeHeader(f)
	require.NoError(t, err)
	_ = a
}

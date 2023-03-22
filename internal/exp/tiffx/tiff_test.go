package tiffx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff"
)

const inflags = os.O_RDONLY
const outflags = os.O_WRONLY | os.O_TRUNC | os.O_CREATE

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

func TestTIFF(t *testing.T) {
	f, err := os.OpenFile("testdata/uncompressed.tiff", inflags, 0)
	defer f.Close()
	// WIP
	require.NoError(t, err)
	img, _ := tiff.Decode(f)
	_ = img
}

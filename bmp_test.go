package imgcut

import (
	"image"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBMP(t *testing.T) {
	inflags := os.O_RDONLY
	outflags := os.O_WRONLY | os.O_TRUNC | os.O_CREATE

	// Convert tiff to bmp
	// f, err := os.OpenFile("testdata/img.tiff", os.O_RDONLY, 0)
	// require.NoError(t, err)
	// a, err := tiff.Decode(f)
	// require.NoError(t, err)
	// out, err := os.OpenFile("testdata/img.bmp", outflags, 0640)
	// err = bmp.Encode(out, a)
	// require.NoError(t, err)
	f, err := os.OpenFile("testdata/img.bmp", inflags, 0)
	require.NoError(t, err)
	bb := NewBMP(f)
	outf, err := os.OpenFile("testdata/cut.bmp", outflags, 0640)
	rect := image.Rectangle{
		image.Point{0, 0},
		image.Point{100, 100},
	}
	err = bb.Cut(rect, outf)
	require.NoError(t, err)
}

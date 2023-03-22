package bmpx

import (
	"fmt"
	"image"
	"io"
	"math/rand"
	"os"
	"testing"

	seekable "github.com/SaveTheRbtz/zstd-seekable-format-go"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/bmp"
)

const inflags = os.O_RDONLY
const outflags = os.O_WRONLY | os.O_TRUNC | os.O_CREATE

func TestBMP(t *testing.T) {
	t.Skip()
	inflags := os.O_RDONLY
	f, err := os.OpenFile("testdata/space.bmp", inflags, 0)
	require.NoError(t, err)
	outf, err := os.OpenFile("testdata/cut.bmp", outflags, 0640)
	rect := image.Rect(24500, 10000, 24900, 11000)
	err = Crop(f, outf, rect)
	outf.Close()
}

func TestBmpZstd(t *testing.T) {
	t.Skip()
	f, err := os.OpenFile("testdata/space.bmp.gz", inflags, 0)
	require.NoError(t, err)
	dec, err := zstd.NewReader(nil)
	require.NoError(t, err)
	r, err := seekable.NewReader(f, dec)
	require.NoError(t, err)

	defer r.Close()
	defer dec.Close()
	defer f.Close()

	outf, err := os.OpenFile("testdata/img2.bmp", outflags, 0644)
	require.NoError(t, err)
	defer outf.Close()

	rect := image.Rect(24500, 10000, 24900, 11000)
	err = Crop(f, io.Discard, rect)
	require.NoError(t, err)
}

func BenchmarkZstd(b *testing.B) {
	b.Skip()
	f, err := os.OpenFile("testdata/img2.bmp.gz", inflags, 0)
	require.NoError(b, err)

	dec, err := zstd.NewReader(nil)
	require.NoError(b, err)
	r, err := seekable.NewReader(f, dec)
	require.NoError(b, err)

	// Read image props
	cfg, err := bmp.DecodeConfig(r)
	require.NoError(b, err)

	width := cfg.Width
	height := cfg.Height

	for dx := 100; dx <= width; dx *= 4 {
		for dy := 100; dy <= height; dy *= 4 {
			b.Run(fmt.Sprintf("{%d,%d}", dx, dy), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					offx := rand.Intn(width - dx)
					offy := rand.Intn(height - dy)
					rect := image.Rect(offx, offy, offx+dx, offy+dy)
					err = Crop(f, io.Discard, rect)
					require.NoError(b, err)
				}
			})
		}
	}
}

func BenchmarkNoCompress(b *testing.B) {
	b.Skip()
	// setup
	inflags := os.O_RDONLY
	f, err := os.OpenFile("testdata/space.bmp", inflags, 0)
	require.NoError(b, err)
	cfg, err := bmp.DecodeConfig(f)
	require.NoError(b, err)
	width := cfg.Width
	height := cfg.Height

	for dx := 100; dx <= width; dx *= 4 {
		for dy := 100; dy <= height; dy *= 4 {
			b.Run(fmt.Sprintf("{%d,%d}", dx, dy), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					f.Seek(0, io.SeekStart)
					offx := rand.Intn(width - dx)
					offy := rand.Intn(height - dy)
					rect := image.Rect(offx, offy, offx+dx, offy+dy)
					err = Crop(f, io.Discard, rect)
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

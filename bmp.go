package imgcut

import (
	"image"
	"io"
	_ "unsafe"

	bmp "golang.org/x/image/bmp"
)

var _ Cutter = new(BMP)

type BMP struct {
	r io.Reader
}

func NewBMP(r io.Reader) *BMP {
	return &BMP{r: r}
}

func (b *BMP) Cut(r image.Rectangle, to io.Writer) error {
	i := bmp.Decode
	_ = i
	a, bb, c, d, e := decodeConfig(b.r)
	_ = a
	_ = bb
	_ = c
	_ = d
	_ = e
	// a, err := bmp.Decode(b.r)
	// if err != nil {
	// 	return nil, err
	// }
	// c := a.At(0, 0)
	// _ = c
	return nil
}

//go:linkname decodeConfig golang.org/x/image/bmp.decodeConfig
func decodeConfig(r io.Reader) (config image.Config, bitsPerPixel int, topDown bool, allowAlpha bool, err error)

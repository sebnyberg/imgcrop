package vipsx

import (
	"errors"
	"image"
	"io"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

type Cropper struct {
	cropCount int
	mtx       sync.Mutex
	r         io.Reader
}

func NewCropper(r io.Reader) *Cropper {
	return &Cropper{r: r}
}

// todo(sn): this is copied from BMP cropper. Not great.
func (c *Cropper) Crop(cropArea image.Rectangle, out io.Writer) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Guard against re-cropping of the same image unless it has been provided
	// as an io.Seeker (can be reset)
	if c.cropCount > 0 {
		s, ok := c.r.(io.Seeker)
		if !ok {
			return errors.New("re-cropping not supported for non-io.Seekers")
		}
		s.Seek(0, io.SeekStart)
	}
	c.cropCount++

	return Crop(c.r, cropArea, out)
}

func Crop(r io.Reader, cropArea image.Rectangle, out io.Writer) error {

	img, err := vips.NewImageFromReader(r)
	if err != nil {
		return err
	}
	x0 := cropArea.Min.X
	y0 := cropArea.Min.Y
	x1 := cropArea.Max.X
	y1 := cropArea.Max.Y
	dx := x1 - x0
	dy := y1 - y0
	err = img.ExtractArea(x0, y0, dx, dy)
	if err != nil {
		return err
	}
	img.ExportTiff(nil)
	return nil
}

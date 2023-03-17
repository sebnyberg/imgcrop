package imgcrop

import (
	"errors"
	"image"
	"io"
	"sync"

	vips "github.com/davidbyttow/govips/v2/vips"
)

type VipsCropper struct {
	cropCount int
	mtx       sync.Mutex
	r         io.Reader
}

func NewVipsCropper(r io.Reader) *VipsCropper {
	return &VipsCropper{r: r}
}

// todo(sn): this is copied from BMP cropper. Not great.
func (c *VipsCropper) Crop(cropArea image.Rectangle, out io.Writer) error {
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

	return VipsCrop(c.r, cropArea, out)
}

func VipsCrop(r io.Reader, cropArea image.Rectangle, out io.Writer) error {
	vips.Startup(nil)
	defer vips.Shutdown()

	img, err := vips.NewImageFromReader(r)
	if err != nil {
		return err
	}
	_ = img
	// var params vips.ExportParams
	// params.

	return nil
}

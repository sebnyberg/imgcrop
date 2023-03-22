package vipsx

import (
	"image"
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

func GoVipsCrop(in io.Reader, out io.Writer, region image.Rectangle) error {
	img, err := vips.NewImageFromReader(in)
	if err != nil {
		return err
	}
	x0 := region.Min.X
	y0 := region.Min.Y
	x1 := region.Max.X
	y1 := region.Max.Y
	dx := x1 - x0
	dy := y1 - y0
	err = img.ExtractArea(x0, y0, dx, dy)
	if err != nil {
		return err
	}
	img.ExportTiff(nil)
	return nil
}

func init() {
	vips.LoggingSettings(nil, vips.LogLevelCritical)
	vips.Startup(nil)
}

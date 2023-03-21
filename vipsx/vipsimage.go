package vipsx

import (
	"image"

	"github.com/vipsimage/vips"
)

func VipsImageCrop(name string, area image.Rectangle, out string) error {
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

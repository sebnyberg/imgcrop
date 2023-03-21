package imgcrop

import (
	"image"
	"io"

	"github.com/sebnyberg/imgcrop/bmpx"
	"github.com/sebnyberg/imgcrop/tiffx"
	"github.com/sebnyberg/imgcrop/vipsx"
)

var _ Cropper = new(bmpx.Cropper)
var _ Cropper = new(tiffx.Cropper)
var _ Cropper = new(vipsx.Cropper)

type Cropper interface {
	// Crop crops the provided region out of an image and puts the result in
	// the provided writer
	Crop(r image.Rectangle, to io.Writer) error
}

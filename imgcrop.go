package imgcut

import (
	"image"
	"io"
)

type Cutter interface {
	// Crop crops the provided region out of an image and puts the result in
	// the provided writer
	Crop(r image.Rectangle, to io.Writer) error
}

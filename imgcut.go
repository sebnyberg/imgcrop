package imgcut

import (
	"image"
	"io"
)

type Cutter interface {
	// Cut cuts the provided rectangle out of an image and puts the result in
	// the provided writer
	Cut(r image.Rectangle, to io.Writer) error
}

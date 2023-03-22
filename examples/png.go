package examples

import (
	"image"
	"image/png"
	"os"

	"github.com/sebnyberg/imgcrop/bmpx"
	"golang.org/x/image/bmp"
)

// cropPNG is not part of the library because it does not meet its memory
// requirements. Converting PNG to BMP and back does a full in-memory
// serialization twice. Why even include this example? Because it shows how
// ineffecient this method is.
func cropPNG(srcPath, dstPath string, region image.Rectangle) error {
	// Decode PNG, encode BMP to temporary file
	src, err := os.OpenFile(srcPath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer src.Close()
	tmpSrcBMP, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer tmpSrcBMP.Close()
	img, err := png.Decode(src)
	if err != nil {
		return err
	}
	err = bmp.Encode(tmpSrcBMP, img)
	if err != nil {
		return err
	}

	// Create temporary cropped BMP output file and crop to it
	tmpDstBMP, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer tmpDstBMP.Close()
	err = bmpx.Crop(src, tmpDstBMP, region)
	if err != nil {
		return err
	}

	// Encode BMP back to PNG
	dst, err := os.OpenFile(dstPath, os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer dst.Close()
	img, err = bmp.Decode(tmpDstBMP)
	if err != nil {
		return err
	}
	return png.Encode(dst, img)
}

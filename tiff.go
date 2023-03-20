package imgcrop

// https://web.archive.org/web/20210108174645/https://www.adobe.io/content/dam/udp/en/open/standards/tiff/TIFF6.pdf
//
// TIFF is a piece of shit standard, but with certain settings, it supports
// uncompressed images and is more widely used than BMP. For example,
// libvips does not support BMP images unless certain extra libraries are
// linked in during compilation. It is certainly not supported by the Go
// wrappers for libvips.
//
// This package imports a subset of the TIFF standard, supporting only
// uncompressed images, a single Image File Directory (IFD), and non-paletted
// images. Anything outside of what this package expects will be violently
// tossed out the window.

import (
	"encoding/binary"
	"errors"
	"io"
)

type tiffDecodeResult struct {
	byteOrder binary.ByteOrder
}

func tiffDecodeHeader(r io.Reader) (res tiffDecodeResult, err error) {
	const (
		leHeader = "II\x2A\x00" // Header for little-endian files.
		beHeader = "MM\x00\x2A" // Header for big-endian files.

		ifdLen = 12 // Length of an IFD entry in bytes.
	)

	var empty tiffDecodeResult
	var b [2048]byte

	if _, err := io.ReadFull(r, b[:8]); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return empty, err
	}

	switch string(b[0:4]) {
	case leHeader:
		res.byteOrder = binary.LittleEndian
	case beHeader:
		res.byteOrder = binary.BigEndian
	default:
		return empty, errors.New("unsupported binary format")
	}

	ifdOffset := int64(res.byteOrder.Uint32(b[4:8]))
	_ = ifdOffset
	return res, errors.New("not totally done, yet")
}

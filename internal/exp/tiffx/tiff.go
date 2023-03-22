package tiffx

import (
	"encoding/binary"
	"errors"
	"io"
)

type DecodeResult struct {
	ByteOrder binary.ByteOrder
}

func DecodeHeader(r io.Reader) (res DecodeResult, err error) {
	const (
		leHeader = "II\x2A\x00" // Header for little-endian files.
		beHeader = "MM\x00\x2A" // Header for big-endian files.

		ifdLen = 12 // Length of an IFD entry in bytes.
	)

	var empty DecodeResult
	var b [1024]byte

	// Read header
	if _, err := io.ReadFull(r, b[:8]); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return empty, err
	}

	switch string(b[0:4]) {
	case leHeader:
		res.ByteOrder = binary.LittleEndian
	case beHeader:
		res.ByteOrder = binary.BigEndian
	default:
		return empty, errors.New("unsupported binary format")
	}

	ifdOffset := int64(res.ByteOrder.Uint32(b[4:8]))
	_ = ifdOffset
	return res, errors.New("not totally done, yet")
}

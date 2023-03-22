package bmpx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path"
)

// CropFile crops the provided region of the BMP found at srcPath to a BMP at
// dstPath. For more info, see Crop().
func CropFile(srcPath, dstPath string, region image.Rectangle) error {
	srcPath = path.Clean(srcPath)
	src, err := os.OpenFile(srcPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open file %q err, %w", srcPath, err)
	}
	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0640)
	if err != nil {
		return fmt.Errorf("open file %q err, %w", dstPath, err)
	}
	return Crop(src, dst, region)
}

// Crop crops the provided region of the BMP found in the input stream to the
// output stream.
//
// The input BMP must be bottom-up, no alpha, and uncompressed.
//
// Thanks to the simplicity of the BMP format, crop uses a very small amount of
// memory (~8KiB).
//
// If src is an io.ReadSeeker, then the cropper will seek to skip pixels that
// are outside the cropping region.
//
// Cropping complexity scales primarily with number of cropped rows, not
// columns. Depending on the data and number of crops, it may make sense to
// rotate the image accordingly.
func Crop(src io.Reader, dst io.Writer, region image.Rectangle) error {
	// Load BMP header bytes and significant content
	hdr, err := DecodeHeader(src)
	if err != nil {
		return err
	}
	if hdr.TopDown {
		return errors.New(".BMP: topDown not supported")
	}
	if hdr.AllowAlpha {
		return errors.New(".BMP: allowAlpha not supported")
	}

	// Find / validate crop area
	dim := image.Rect(0, 0, hdr.Config.Width, hdr.Config.Height)
	region = dim.Intersect(region)
	if region.Empty() {
		return errors.New("crop area empty or out of bounds")
	}

	// Create updated BMP header with crop dimensions
	totalSize := (hdr.BitsPerPixel/8)*(region.Dx()*region.Dy()) + len(hdr.HeaderBytes)
	width := region.Dx()
	height := region.Dy()
	binary.LittleEndian.PutUint32(hdr.HeaderBytes[2:6], uint32(totalSize))
	binary.LittleEndian.PutUint32(hdr.HeaderBytes[18:22], uint32(width))
	binary.LittleEndian.PutUint32(hdr.HeaderBytes[22:26], uint32(height))
	_, err = dst.Write(hdr.HeaderBytes)
	if err != nil {
		return err
	}

	bytesPerPixel := hdr.BitsPerPixel / 8

	// Seek if possible, otherwise copy to discard
	var seek func(off int) (n int64, err error)
	if s, ok := src.(io.Seeker); ok {
		seek = func(off int) (n int64, err error) {
			return s.Seek(int64(off), io.SeekCurrent)
		}
	} else {
		seek = func(off int) (n int64, err error) {
			return io.CopyN(io.Discard, src, int64(off))
		}
	}

	byteWidth := func(pixels, bitsPerPixel int) int {
		return ((pixels*bitsPerPixel + 31) / 32) * 4
	}

	// Skip uncropped last rows (recall: bmp is bottom-up in this case)
	rowBytes := byteWidth(hdr.BitsPerPixel, hdr.Config.Width)
	skipBytes := rowBytes * (hdr.Config.Height - region.Max.Y)
	if _, err := seek(skipBytes); err != nil {
		return err
	}

	// Now within cropping region in terms of y
	// There are some nuances to be aware of: each BMP pixel row is padded to be
	// 4-byte aligned. This means that there may be extra bytes that are empty
	// on each row that is being read, and that padding may need to be added to
	// the row that is being written.
	left := bytesPerPixel * region.Min.X
	mid := region.Dx() * bytesPerPixel
	right := rowBytes - (mid + left)
	wantWidth := byteWidth(hdr.BitsPerPixel, region.Dx())
	padding := make([]byte, wantWidth-mid)

	for dy := 1; dy <= region.Dy(); dy++ {
		// Skip left
		_, err := seek(left)
		if err != nil {
			return err
		}

		// Write middle part with padding
		n, err := io.CopyN(dst, src, int64(mid))
		if err != nil || n != int64(mid) {
			return err
		}
		_, err = dst.Write(padding)
		if err != nil {
			return err
		}

		// Skip right
		_, err = seek(right)
		if err != nil {
			return err
		}
	}

	return nil
}

type DecodeResult struct {
	Config       image.Config
	BitsPerPixel int
	TopDown      bool
	AllowAlpha   bool
	HeaderBytes  []byte
	ImageOffset  uint32
}

// bmpDecodeHeader was shamelessly copied from 'x/image/bmp' and edited for the
// usecase in this repo. Unlike the stdlib implementation, the header and
// palette bytes are retained so that they can be re-written to cropped images.
func DecodeHeader(r io.Reader) (res DecodeResult, err error) {
	readUint16 := func(b []byte) uint16 {
		return uint16(b[0]) | uint16(b[1])<<8
	}
	readUint32 := func(b []byte) uint32 {
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	}

	// We only support those BMP images with one of the following DIB headers:
	// - BITMAPINFOHEADER (40 bytes)
	// - BITMAPV4HEADER (108 bytes)
	// - BITMAPV5HEADER (124 bytes)
	const (
		fileHeaderLen   = 14
		infoHeaderLen   = 40
		v4InfoHeaderLen = 108
		v5InfoHeaderLen = 124
	)
	var empty DecodeResult
	var b [2048]byte
	if _, err := io.ReadFull(r, b[:fileHeaderLen+4]); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return empty, err
	}
	if string(b[:2]) != "BM" {
		return empty, errors.New("bmp: invalid format")
	}
	offset := readUint32(b[10:14])
	res.ImageOffset = offset
	res.HeaderBytes = b[:offset]
	infoLen := readUint32(b[14:18])
	if infoLen != infoHeaderLen && infoLen != v4InfoHeaderLen && infoLen != v5InfoHeaderLen {
		return empty, errors.New("unsupported")
	}
	if _, err := io.ReadFull(r, b[fileHeaderLen+4:fileHeaderLen+infoLen]); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return empty, err
	}
	width := int(int32(readUint32(b[18:22])))
	height := int(int32(readUint32(b[22:26])))
	if height < 0 {
		height, res.TopDown = -height, true
	}
	if width < 0 || height < 0 {
		return empty, errors.New("unsupported")
	}
	// We only support 1 plane and 8, 24 or 32 bits per pixel and no
	// compression.
	planes, bpp, compression := readUint16(b[26:28]), readUint16(b[28:30]), readUint32(b[30:34])
	// if compression is set to BI_BITFIELDS, but the bitmask is set to the default bitmask
	// that would be used if compression was set to 0, we can continue as if compression was 0
	if compression == 3 && infoLen > infoHeaderLen &&
		readUint32(b[54:58]) == 0xff0000 && readUint32(b[58:62]) == 0xff00 &&
		readUint32(b[62:66]) == 0xff && readUint32(b[66:70]) == 0xff000000 {
		compression = 0
	}
	if planes != 1 || compression != 0 {
		return empty, errors.New("unsupported")
	}
	switch bpp {
	case 8:
		if offset != fileHeaderLen+infoLen+256*4 {
			return empty, errors.New("unsupported")
		}
		pre := fileHeaderLen + int(infoLen)
		_, err = io.ReadFull(r, b[pre:pre+256*4])
		if err != nil {
			return empty, err
		}
		pcm := make(color.Palette, 256)
		for i := range pcm {
			// BMP images are stored in BGR order rather than RGB order.
			// Every 4th byte is padding.
			pcm[i] = color.RGBA{b[pre+4*i+2], b[pre+4*i+1], b[pre+4*i+0], 0xFF}
		}
		res.Config = image.Config{ColorModel: pcm, Width: width, Height: height}
		res.BitsPerPixel = 8
		res.AllowAlpha = false
		return res, nil
	case 24:
		if offset != fileHeaderLen+infoLen {
			return empty, errors.New("unsupported")
		}
		res.Config = image.Config{ColorModel: color.RGBAModel, Width: width, Height: height}
		res.BitsPerPixel = 24
		res.AllowAlpha = false
		return res, nil
	case 32:
		if offset != fileHeaderLen+infoLen {
			return empty, errors.New("unsupported")
		}
		// 32 bits per pixel is possibly RGBX (X is padding) or RGBA (A is
		// alpha transparency). However, for BMP images, "Alpha is a
		// poorly-documented and inconsistently-used feature" says
		// https://source.chromium.org/chromium/chromium/src/+/bc0a792d7ebc587190d1a62ccddba10abeea274b:third_party/blink/renderer/platform/image-decoders/bmp/bmp_image_reader.cc;l=621
		//
		// That goes on to say "BITMAPV3HEADER+ have an alpha bitmask in the
		// info header... so we respect it at all times... [For earlier
		// (smaller) headers we] ignore alpha in Windows V3 BMPs except inside
		// ICO files".
		//
		// "Ignore" means to always set alpha to 0xFF (fully opaque):
		// https://source.chromium.org/chromium/chromium/src/+/bc0a792d7ebc587190d1a62ccddba10abeea274b:third_party/blink/renderer/platform/image-decoders/bmp/bmp_image_reader.h;l=272
		//
		// Confusingly, "Windows V3" does not correspond to BITMAPV3HEADER, but
		// instead corresponds to the earlier (smaller) BITMAPINFOHEADER:
		// https://source.chromium.org/chromium/chromium/src/+/bc0a792d7ebc587190d1a62ccddba10abeea274b:third_party/blink/renderer/platform/image-decoders/bmp/bmp_image_reader.cc;l=258
		//
		// This Go package does not support ICO files and the (infoLen >
		// infoHeaderLen) condition distinguishes BITMAPINFOHEADER (40 bytes)
		// vs later (larger) headers.
		res.AllowAlpha = infoLen > infoHeaderLen
		res.Config = image.Config{ColorModel: color.RGBAModel, Width: width, Height: height}
		res.BitsPerPixel = 32
		res.AllowAlpha = infoLen > infoHeaderLen
		return res, nil
	}
	return empty, errors.New("unsupported")
}

package imgcut

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"sync"
	_ "unsafe"

	"golang.org/x/image/bmp"
)

var _ Cropper = new(BMPMultiCropper)
var _ any = bmp.Decode

type BMPMultiCropper struct {
	cropCount int
	mtx       sync.Mutex
	r         io.Reader
}

func NewBMPMultiCropper(r io.Reader) *BMPMultiCropper {
	return &BMPMultiCropper{r: r}
}

func (b *BMPMultiCropper) Crop(cropArea image.Rectangle, out io.Writer) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// Guard against re-cropping of the same image unless it has been provided
	// as an io.Seeker (can be reset)
	if b.cropCount > 0 {
		s, ok := b.r.(io.Seeker)
		if !ok {
			return errors.New("re-cropping not supported for non-io.Seekers")
		}
		s.Seek(0, io.SeekStart)
	}
	b.cropCount++

	return BMPCrop(b.r, cropArea, out)
}

func BMPCrop(r io.Reader, cropArea image.Rectangle, out io.Writer) error {

	// Load BMP header bytes and significant content
	hdr, err := decodeConfig(b.r)
	if err != nil {
		return err
	}
	if hdr.topDown {
		return errors.New(".BMP: topDown not supported")
	}
	if hdr.allowAlpha {
		return errors.New(".BMP: allowAlpha not supported")
	}

	// Find / validate crop area
	dim := image.Rect(0, 0, hdr.config.Width, hdr.config.Height)
	cropArea = dim.Intersect(cropArea)
	if cropArea.Empty() {
		return errors.New("crop area empty or out of bounds")
	}

	// Create updated BMP header with crop dimensions
	totalSize := (hdr.bitsPerPixel/8)*(cropArea.Dx()*cropArea.Dy()) + len(hdr.header)
	width := cropArea.Dx()
	height := cropArea.Dy()
	binary.LittleEndian.PutUint32(hdr.header[2:6], uint32(totalSize))
	binary.LittleEndian.PutUint32(hdr.header[18:22], uint32(width))
	binary.LittleEndian.PutUint32(hdr.header[22:26], uint32(height))
	_, err = out.Write(hdr.header)
	if err != nil {
		return err
	}

	bytesPerPixel := hdr.bitsPerPixel / 8

	// Seek if possible, otherwise copy to discard
	var seek func(off int) (n int64, err error)
	if s, ok := b.r.(io.Seeker); ok {
		seek = func(off int) (n int64, err error) {
			return s.Seek(int64(off), io.SeekCurrent)
		}
	} else {
		seek = func(off int) (n int64, err error) {
			return io.CopyN(io.Discard, b.r, int64(off))
		}
	}

	byteWidth := func(pixels, bitsPerPixel int) int {
		return ((pixels*bitsPerPixel + 31) / 32) * 4
	}

	// Skip uncropped last rows (recall: bmp is bottom-up in this case)
	rowBytes := byteWidth(hdr.bitsPerPixel, hdr.config.Width)
	skipBytes := rowBytes * (hdr.config.Height - cropArea.Max.Y)
	if _, err := seek(skipBytes); err != nil {
		return err
	}

	// Now within cropping region in terms of y
	// There are some nuances to be aware of: each BMP pixel row is padded to be
	// 4-byte aligned. This means that there may be extra bytes that are empty
	// on each row that is being read, and that padding may need to be added to
	// the row that is being written.
	left := bytesPerPixel * cropArea.Min.X
	mid := cropArea.Dx() * bytesPerPixel
	right := rowBytes - (mid + left)
	wantWidth := byteWidth(hdr.bitsPerPixel, cropArea.Dx())
	padding := make([]byte, wantWidth-mid)

	for dy := 1; dy <= cropArea.Dy(); dy++ {
		// Skip left
		_, err := seek(left)
		if err != nil {
			return err
		}

		// Write middle part with padding
		n, err := io.CopyN(out, b.r, int64(mid))
		if err != nil || n != int64(mid) {
			return err
		}
		_, err = out.Write(padding)
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

func readUint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func readUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

type decodeResult struct {
	config       image.Config
	bitsPerPixel int
	topDown      bool
	allowAlpha   bool
	header       []byte
	offset       uint32
}

// decodeConfig was shamelessly copied from 'x/image/bmp' and edited for the
// usecase in this repo. Unlike the stdlib implementation, the header and
// palette bytes are retained so that they can be re-written to cropped images.
func decodeConfig(r io.Reader) (res decodeResult, err error) {
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
	var empty decodeResult
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
	res.offset = offset
	res.header = b[:offset]
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
		height, res.topDown = -height, true
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
		res.config = image.Config{ColorModel: pcm, Width: width, Height: height}
		res.bitsPerPixel = 8
		res.allowAlpha = false
		return res, nil
	case 24:
		if offset != fileHeaderLen+infoLen {
			return empty, errors.New("unsupported")
		}
		res.config = image.Config{ColorModel: color.RGBAModel, Width: width, Height: height}
		res.bitsPerPixel = 24
		res.allowAlpha = false
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
		res.allowAlpha = infoLen > infoHeaderLen
		res.config = image.Config{ColorModel: color.RGBAModel, Width: width, Height: height}
		res.bitsPerPixel = 32
		res.allowAlpha = infoLen > infoHeaderLen
		return res, nil
	}
	return empty, errors.New("unsupported")
}

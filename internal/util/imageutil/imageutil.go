package imageutil

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"

	"golang.org/x/image/draw"
)

// DecodeImage decodes an image from the given file path.
// For JPEG files, EXIF orientation is read and applied automatically.
func DecodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	orient, err := readJPEGOrientation(path)
	if err == nil {
		img = applyOrientation(img, orient)
	}

	return img, nil
}

// ResizeImage resizes the image to the given width, preserving aspect ratio.
// If height is 0, it is calculated from the original aspect ratio.
func ResizeImage(src image.Image, width int) image.Image {
	if width <= 0 {
		return src
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	height := int(float64(srcH) * float64(width) / float64(srcW))
	if height < 1 {
		height = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func readJPEGOrientation(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseJPEGOrientation(data)
}

const (
	// EXIF marker types
	markerAPP1    = 0xE1
	markerSOI     = 0xD8
	markerSOS     = 0xDA
	markerEOI     = 0xD9
	markerDQT     = 0xDB
	markerDHT     = 0xC4
	markerSOF0    = 0xC0 // Baseline DCT
	markerSOF1    = 0xC1
	markerSOF2    = 0xC2
	markerSOF3    = 0xC3
	markerSOF5    = 0xC5
	markerSOF6    = 0xC6
	markerSOF7    = 0xC7
	markerSOF9    = 0xC9
	markerSOF10   = 0xCA
	markerSOF11   = 0xCB
	markerSOF13   = 0xCD
	markerSOF14   = 0xCE
	markerSOF15   = 0xCF
	markerRST0    = 0xD0 // RSTn markers
	markerRST7    = 0xD7
	markerTEM     = 0x01
)

func parseJPEGOrientation(data []byte) (int, error) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != markerSOI {
		return 0, fmt.Errorf("not a JPEG file")
	}

	pos := 2
	for pos < len(data) {
		// Skip padding 0xFF bytes
		for pos < len(data) && data[pos] == 0xFF {
			pos++
		}
		if pos >= len(data) {
			break
		}

		marker := data[pos]
		pos++

		// Entropy-coded data markers have no length field; stop parsing
		if marker == markerSOS {
			break
		}

		// Skip standalone markers
		switch marker {
		case markerSOI, markerEOI, markerTEM, markerRST0, markerRST7:
			continue
		}

		if pos+2 > len(data) {
			break
		}

		length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2

		if length < 2 {
			break
		}

		segmentEnd := pos + length - 2
		if segmentEnd > len(data) {
			break
		}

		if marker == markerAPP1 && pos+6 <= segmentEnd {
			if string(data[pos:pos+6]) == "Exif\x00\x00" {
				orient, err := parseEXIFOrientation(data[pos+6 : segmentEnd])
				if err == nil {
					return orient, nil
				}
				return 0, err
			}
		}

		pos = segmentEnd
	}

	return 0, fmt.Errorf("no EXIF orientation found")
}

func parseEXIFOrientation(data []byte) (int, error) {
	if len(data) < 8 {
		return 0, fmt.Errorf("EXIF data too short")
	}

	var byteOrder binary.ByteOrder
	byteOrder = binary.LittleEndian
	if data[0] == 'M' && data[1] == 'M' {
		byteOrder = binary.BigEndian
	} else if data[0] == 'I' && data[1] == 'I' {
		byteOrder = binary.LittleEndian
	} else {
		return 0, fmt.Errorf("invalid TIFF byte order")
	}

	magic := byteOrder.Uint16(data[2:4])
	if magic != 0x002A {
		return 0, fmt.Errorf("invalid TIFF magic")
	}

	ifdOffset := int(byteOrder.Uint32(data[4:8]))
	return parseIFD(data, ifdOffset, byteOrder)
}

func parseIFD(data []byte, offset int, byteOrder binary.ByteOrder) (int, error) {
	if offset+2 > len(data) {
		return 0, fmt.Errorf("IFD out of bounds")
	}

	numEntries := int(byteOrder.Uint16(data[offset : offset+2]))
	entryOffset := offset + 2

	const orientationTag = 0x0112
	const entrySize = 12

	for i := 0; i < numEntries; i++ {
		entryStart := entryOffset + i*entrySize
		if entryStart+entrySize > len(data) {
			break
		}

		entry := data[entryStart : entryStart+entrySize]
		tag := byteOrder.Uint16(entry[0:2])

		if tag == orientationTag {
			tagType := byteOrder.Uint16(entry[2:4])
			if tagType != 3 { // SHORT
				return 0, fmt.Errorf("orientation tag has unexpected type %d", tagType)
			}
			orient := int(byteOrder.Uint16(entry[8:10]))
			return orient, nil
		}
	}

	// Check for next IFD offset
	nextIFDOffsetStart := entryOffset + numEntries*entrySize
	if nextIFDOffsetStart+4 > len(data) {
		return 0, fmt.Errorf("orientation tag not found in IFD")
	}

	nextIFDOffset := int(byteOrder.Uint32(data[nextIFDOffsetStart : nextIFDOffsetStart+4]))
	if nextIFDOffset == 0 {
		return 0, fmt.Errorf("orientation tag not found")
	}

	return parseIFD(data, nextIFDOffset, byteOrder)
}

// applyOrientation transforms the image according to EXIF orientation tag.
// Valid orientations: 1-8
func applyOrientation(src image.Image, orientation int) image.Image {
	switch orientation {
	case 1:
		return src
	case 2:
		return flipHorizontal(src)
	case 3:
		return rotate180(src)
	case 4:
		return flipVertical(src)
	case 5:
		return flipHorizontal(rotate270(src))
	case 6:
		return rotate90(src)
	case 7:
		return flipHorizontal(rotate90(src))
	case 8:
		return rotate270(src)
	default:
		return src
	}
}

func flipHorizontal(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func flipVertical(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, h-1-y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func rotate90(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, h-1-y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func rotate270(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(y, w-1-x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

package util

import (
	"encoding/binary"
	"io"
	"os"
)

// mp4TagNames maps the raw MP4/QuickTime item atom types we care about to a
// stable, human-readable key. The first four bytes of a QuickTime metadata key
// can be a 0xA9 copyright byte, hence the escaped literals.
var mp4TagNames = map[string]string{
	"\xa9cmt": "comment",          // ©cmt
	"desc":    "description",      // desc
	"\xa9nam": "title",            // ©nam
	"\xa9day": "date",             // ©day
	"\xa9too": "encoder",          // ©too
	"ldes":    "long_description", // ldes
}

// mp4ContainerBoxes are boxes whose payload is a sequence of child boxes. We
// only recurse into the boxes that can contain the metadata we are after so
// large payload boxes (mdat, stbl tables, ...) are skipped by seeking.
var mp4ContainerBoxes = map[string]bool{
	"moov": true,
	"udta": true,
	"ilst": true,
}

const mp4BoxHeaderSize = 8

// ReadMP4Tags extracts embedded descriptive tags (description, comment, title,
// date, ...) from an MP4/MOV file. It parses the box tree up to and including
// moov/udta/meta/ilst and never reads the mdat payload, so the cost is
// proportional to the moov size (tens of KB), not the file size.
//
// A missing or unparseable file returns an error; a valid file with no tags
// returns an empty (non-nil) map.
func ReadMP4Tags(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	if err := scanMP4Boxes(f, stat.Size(), tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// ReadMP4Tag returns the value of a single tag (e.g. "description" or
// "comment"), or ("", false) when it is absent.
func ReadMP4Tag(path, key string) (string, bool) {
	tags, err := ReadMP4Tags(path)
	if err != nil {
		return "", false
	}
	v, ok := tags[key]
	return v, ok
}

// scanMP4Boxes walks the box tree in [start, end) recording recognised tag
// items into out. It tolerates truncated/odd tail data and 64-bit box sizes.
func scanMP4Boxes(r io.ReadSeeker, end int64, out map[string]string) error {
	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		if pos+mp4BoxHeaderSize > end {
			return nil
		}

		var header [mp4BoxHeaderSize]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			// Truncated or malformed tail: stop rather than fail the whole read.
			return nil
		}

		size := int64(binary.BigEndian.Uint32(header[0:4]))
		boxType := string(header[4:8])

		switch size {
		case 1:
			// 64-bit extended size follows the header.
			var large [8]byte
			if _, err := io.ReadFull(r, large[:]); err != nil {
				return nil
			}
			size = int64(binary.BigEndian.Uint64(large[:]))
		case 0:
			// Box extends to the end of the enclosing container.
			size = end - pos
		}

		headerLen := int64(mp4BoxHeaderSize)
		if binary.BigEndian.Uint32(header[0:4]) == 1 {
			headerLen = 16
		}
		if size < headerLen || pos+size > end {
			return nil
		}
		boxEnd := pos + size

		switch {
		case mp4TagNames[boxType] != "":
			if value, ok := readMP4DataBox(r, boxEnd); ok {
				out[mp4TagNames[boxType]] = value
			}
		case boxType == "meta":
			// meta is a full box: skip the 4-byte version/flags before children.
			if _, err := r.Seek(4, io.SeekCurrent); err != nil {
				return err
			}
			if err := scanMP4Boxes(r, boxEnd, out); err != nil {
				return err
			}
		case mp4ContainerBoxes[boxType]:
			if err := scanMP4Boxes(r, boxEnd, out); err != nil {
				return err
			}
		}

		if _, err := r.Seek(boxEnd, io.SeekStart); err != nil {
			return nil
		}
	}
}

// readMP4DataBox reads the first `data` child box in [start, end) and returns
// its UTF-8 payload. A metadata item wraps exactly one data box.
func readMP4DataBox(r io.ReadSeeker, end int64) (string, bool) {
	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return "", false
		}
		if pos+mp4BoxHeaderSize > end {
			return "", false
		}

		var header [mp4BoxHeaderSize]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return "", false
		}

		size := int64(binary.BigEndian.Uint32(header[0:4]))
		boxType := string(header[4:8])
		if size < mp4BoxHeaderSize || pos+size > end {
			return "", false
		}

		if boxType == "data" {
			// data box: 4-byte type indicator + 4-byte locale + payload.
			if size < 16 {
				return "", false
			}
			var dataHeader [8]byte
			if _, err := io.ReadFull(r, dataHeader[:]); err != nil {
				return "", false
			}
			payloadLen := size - 16
			payload := make([]byte, payloadLen)
			if _, err := io.ReadFull(r, payload); err != nil {
				return "", false
			}
			return string(payload), true
		}

		if _, err := r.Seek(pos+size, io.SeekStart); err != nil {
			return "", false
		}
	}
}

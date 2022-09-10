package asar

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"strconv"

	"github.com/go-extras/errors"
	_ "github.com/go-extras/errors"
)

type entryEncoder struct {
	Contents      []io.Reader
	CurrentOffset int64
	Header        bytes.Buffer
	Encoder       *json.Encoder
}

func (enc *entryEncoder) Write(v interface{}) {
	enc.Encoder.Encode(v)
	enc.Header.Truncate(enc.Header.Len() - 1) // cut off trailing new line
}

func (enc *entryEncoder) WriteField(key string, v interface{}) {
	enc.Write(key)
	enc.Header.WriteByte(':')
	enc.Write(v)
}

func (enc *entryEncoder) Encode(e *Entry) error {
	enc.Header.WriteByte('{')
	if e.Flags&FlagDir != 0 {
		if e.Flags&FlagUnpacked != 0 {
			enc.WriteField("unpacked", true)
			enc.Header.WriteByte(',')
		}
		enc.Write("files")
		enc.Header.WriteString(":{")
		for i, child := range e.Children {
			if i > 0 {
				enc.Header.WriteByte(',')
			}
			if !validFilename(child.Name) {
				panic(errors.Wrapf(errHeader, "!validFileName(%s)", child.Name))
			}
			enc.Write(child.Name)
			enc.Header.WriteByte(':')
			if err := enc.Encode(child); err != nil {
				return errors.Wrap(err, "Failed to encode")
			}
		}
		enc.Header.WriteByte('}')
	} else {
		enc.Write("size")
		enc.Header.WriteByte(':')
		enc.Write(e.Size)

		if e.Flags&FlagExecutable != 0 {
			enc.Header.WriteByte(',')
			enc.WriteField("executable", true)
		}

		enc.Header.WriteByte(',')
		if e.Flags&FlagUnpacked == 0 {
			enc.WriteField("offset", strconv.FormatInt(enc.CurrentOffset, 10))
			enc.CurrentOffset += e.Size
			enc.Contents = append(enc.Contents, io.NewSectionReader(e.r, e.Offset+e.baseOffset, e.Size))
		} else {
			enc.WriteField("unpacked", true)
		}
	}
	enc.Header.WriteByte('}')
	return nil
}

// EncodeTo writes an ASAR archive containing Entry's descendants. This function
// is usually called on the root entry.
func (e *Entry) EncodeTo(w io.Writer) (n int64, err error) {

	defer func() {
		if r := recover(); r != nil {
			if e := r.(error); e != nil {
				err = errors.Wrap(e, "from panic")
			} else {
				panic(r)
			}
		}

	}()

	encoder := entryEncoder{}
	{
		var reserve [16]byte
		encoder.Header.Write(reserve[:])
	}
	encoder.Encoder = json.NewEncoder(&encoder.Header)
	if err = encoder.Encode(e); err != nil {
		return 0, errors.Wrap(err, "failed to encode")
	}

	length := encoder.Header.Len() - 16
	var newLen int
	{
		var padding [3]byte
		if mod := length % 4; mod != 0 {
			encoder.Header.Write(padding[:4-mod])
		}
		newLen = encoder.Header.Len() - 16
	}

	header := encoder.Header.Bytes()
	binary.LittleEndian.PutUint32(header[:4], 4)
	binary.LittleEndian.PutUint32(header[4:8], 8+uint32(newLen))
	binary.LittleEndian.PutUint32(header[8:12], 4+uint32(newLen))
	binary.LittleEndian.PutUint32(header[12:16], uint32(length))

	n, err = encoder.Header.WriteTo(w)
	if err != nil {
		return n, errors.Wrap(err, "failed to Header.WriteTo")
	}

	for _, chunk := range encoder.Contents {
		var written int64
		written, err = io.Copy(w, chunk)
		n += written
		if err != nil {
			return n, errors.Wrap(err, "failed to io.Copy")
		}
	}

	return n, nil
}

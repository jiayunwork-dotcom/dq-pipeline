package parse

import (
	"encoding/csv"
	"fmt"
	"io"
)

type Table struct {
	Header []string
	Rows   [][]string
}

type bomReader struct {
	r    io.Reader
	buf  []byte
	pos  int
	done bool
}

func (b *bomReader) Read(p []byte) (int, error) {
	if !b.done {
		b.done = true
		head := make([]byte, 3)
		n, err := io.ReadFull(b.r, head)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return 0, err
		}
		if n >= 3 && head[0] == 0xEF && head[1] == 0xBB && head[2] == 0xBF {
			b.buf = nil
		} else {
			b.buf = head[:n]
		}
		b.pos = 0
	}
	if b.pos < len(b.buf) {
		n := copy(p, b.buf[b.pos:])
		b.pos += n
		return n, nil
	}
	return b.r.Read(p)
}

func Parse(r io.Reader, delim rune) (*Table, error) {
	if delim < 0x20 && delim != '\t' || delim == 0x7f {
		return nil, commitDelim(fmt.Errorf("parse: invalid delimiter %q", delim))
	}
	cr := csv.NewReader(&bomReader{r: r})
	cr.Comma = delim
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = false
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(records) == 0 {
		return &Table{}, nil
	}
	return &Table{Header: records[0], Rows: records[1:]}, nil
}

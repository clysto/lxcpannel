package common

import (
	"io"
	"strings"
)

type InterruptibleReader struct {
	dataChan chan []byte
	errChan  chan error
	buf      []byte
}

func NewInterruptibleReader(reader io.Reader) *InterruptibleReader {
	ir := &InterruptibleReader{
		dataChan: make(chan []byte, 1024),
		errChan:  make(chan error),
		buf:      nil,
	}
	go ir.readFromReader(reader)
	return ir
}

func (r *InterruptibleReader) readFromReader(reader io.Reader) {
	defer close(r.dataChan)
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		r.dataChan <- data
	}
}

func (r *InterruptibleReader) Read(p []byte) (n int, err error) {
	if r.buf != nil {
		if len(r.buf) > len(p) {
			copy(p, r.buf[:len(p)])
			r.buf = r.buf[len(p):]
			return len(p), nil
		}
		copy(p, r.buf)
		n = len(r.buf)
		r.buf = nil
		return n, nil
	}
	select {
	case data, ok := <-r.dataChan:
		if !ok {
			return 0, io.EOF
		}
		if len(data) > len(p) {
			r.buf = data[len(p):]
			data = data[:len(p)]
		}
		copy(p, data)
		return len(data), nil
	case err := <-r.errChan:
		return 0, err
	}
}

func (r *InterruptibleReader) SendEOF() {
	r.errChan <- io.EOF
}

func WordWrap(text string, lineWidth int) string {
	var result strings.Builder
	var currentLineLength int

	for i := 0; i < len(text); i++ {
		char := text[i]

		if char == '\n' {
			result.WriteByte(char)
			currentLineLength = 0
			continue
		}

		if currentLineLength == lineWidth {
			result.WriteByte('\n')
			currentLineLength = 0
		}

		result.WriteByte(char)
		currentLineLength++
	}

	return result.String()
}

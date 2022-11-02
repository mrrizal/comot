package utils

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gosuri/uiprogress"
)

type writer struct {
	w             io.WriterAt
	off           int
	CounterStream chan CounterStream
	id            int
}

type WriteCounter struct {
	Total uint
}

type CounterStream struct {
	ID   int
	Data int
}

type LimitOffsetData struct {
	Limit  int
	Offset int
}

func NewWriter(w io.WriterAt, off, writerID int, counterStream chan CounterStream) *writer {
	return &writer{w, off, counterStream, writerID}
}

func (w *writer) Write(p []byte) (n int, err error) {
	n, err = w.w.WriteAt(p, int64(w.off))
	w.off += n
	w.CounterStream <- CounterStream{w.id, n}
	return
}

func CreateFile(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	return f, err
}

func CountLimitOffset(contentLength, concurrent int) map[int]LimitOffsetData {
	result := make(map[int]LimitOffsetData)
	chunkSize := contentLength / concurrent
	counter := 1
	for i := 0; i < contentLength; i += chunkSize {
		offset := i
		limit := i + chunkSize
		if limit > contentLength {
			limit = contentLength
		}
		result[counter] = LimitOffsetData{Offset: offset, Limit: limit}
		counter++
	}
	return result
}

func SetupProgressBar(concurrent int, limitOffsetData map[int]LimitOffsetData) []*uiprogress.Bar {
	uiprogress.Start()
	bars := make([]*uiprogress.Bar, concurrent)
	for key, value := range limitOffsetData {
		barProgress := func(key int) *uiprogress.Bar {
			startTime := time.Now()
			bar := uiprogress.AddBar(value.Limit - value.Offset).AppendCompleted().PrependFunc(
				func(b *uiprogress.Bar) string {
					second := time.Since(startTime).Seconds()
					downloadSpeed := float64(b.Current()) / second / 1024
					return fmt.Sprintf("%s %.1f kbps", fmt.Sprintf("worker %d", key), downloadSpeed)
				})
			return bar
		}
		bars[key-1] = barProgress(key)
	}
	return bars
}

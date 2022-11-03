package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	counter := 0
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
					return fmt.Sprintf("%s %.1f kbps", fmt.Sprintf("worker %d", key+1), downloadSpeed)
				})
			return bar
		}
		bars[key] = barProgress(key)
	}
	return bars
}

func DeletePartFile() error {
	files, err := filepath.Glob("*.part")
	if err != nil {
		return err
	}

	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func WritePartFile(tracker map[int]int, filename string, limitfOffsetData map[int]LimitOffsetData) error {
	for key, value := range tracker {
		newFilename := fmt.Sprintf("%s_%d.part", filename, key)
		f, err := CreateFile(newFilename)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.WriteString(fmt.Sprintf("%s,%s", strconv.Itoa(value), strconv.Itoa(limitfOffsetData[key].Offset)))
		if err != nil {
			return err
		}
	}
	return nil
}

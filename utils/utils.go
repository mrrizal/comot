package utils

import (
	"io"
	"os"

	"github.com/gosuri/uiprogress"
)

type writer struct {
	w   io.WriterAt
	off int64
}

type WriteCounter struct {
	Total uint64
	Bar   *uiprogress.Bar
}

func NewWriter(w io.WriterAt, off int64) *writer {
	return &writer{w, off}
}

func (w *writer) Write(p []byte) (n int, err error) {
	n, err = w.w.WriteAt(p, w.off)
	w.off += int64(n)
	return
}

func CreateFile(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	return f, err
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.Bar.Set(int(wc.Total))
	return n, nil
}

func ProgressBarSetup(offset, limit int64, name string) *WriteCounter {
	steps := limit - offset
	counter := &WriteCounter{}
	counter.Bar = uiprogress.AddBar(int(steps)).PrependCompleted().AppendCompleted()
	counter.Bar.Total = int(steps)
	counter.Bar.PrependFunc(func(b *uiprogress.Bar) string {
		return name
	})
	return counter
}

package utils

import (
	"fmt"
	"io"
	"os"
	"time"

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
	counter.Bar = uiprogress.AddBar(int(steps))
	counter.Bar.AppendCompleted()
	counter.Bar.Total = int(steps)
	startTime := time.Now()
	counter.Bar.PrependFunc(func(b *uiprogress.Bar) string {
		second := time.Now().Sub(startTime).Seconds()
		downloadSpeed := float64(b.Current()) / second / 1024
		return fmt.Sprintf("%s %.1f kbps", name, downloadSpeed)
	})
	return counter
}

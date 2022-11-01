package utils

import (
	"io"
	"os"
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

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint(n)
	return n, nil
}

func CounterSetup(offset, limit int, name string) *WriteCounter {
	// steps := limit - offset
	counter := &WriteCounter{}
	// counter.Bar = uiprogress.AddBar(int(steps))
	// counter.Bar.AppendCompleted()
	// counter.Bar.Total = int(steps)
	// startTime := time.Now()
	// counter.Bar.PrependFunc(func(b *uiprogress.Bar) string {
	// 	second := time.Since(startTime).Seconds()
	// 	downloadSpeed := float64(b.Current()) / second / 1024
	// 	return fmt.Sprintf("%s %.1f kbps", name, downloadSpeed)
	// })
	return counter
}

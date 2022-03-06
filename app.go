package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/gosuri/uiprogress"
	"github.com/mrrizal/comot/utils"
	"golang.org/x/sync/errgroup"
)

func worker(workerID int64, w io.WriterAt, urlInput string, off, limit int64) error {
	counter := utils.ProgressBarSetup(off, limit, fmt.Sprintf("worker %d:", workerID))

	req, err := http.NewRequest("GET", urlInput, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, limit))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusPartialContent {
		err = fmt.Errorf("server responded with %d status code, expected %d", resp.StatusCode,
			http.StatusPartialContent)
		return err
	}

	_, err = io.Copy(utils.NewWriter(w, off), io.TeeReader(resp.Body, counter))
	return err
}

func is_valid_url(urlInput string) (*http.Response, error) {
	resp, err := http.Head(urlInput)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server responsed with %d status code, expected %d", resp.StatusCode, http.StatusOK)
	}

	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return nil, errors.New("server doest not support range requests")
	}

	return resp, nil
}

func comot(urlInput string, concurent int64) {
	resp, err := is_valid_url(urlInput)
	if err != nil {
		log.Fatal(err.Error())
	}

	contentLength := resp.ContentLength
	if contentLength <= 0 {
		log.Fatal(errors.New("server sent invalid Content-Length Header"))
	}

	filename := path.Base(resp.Request.URL.Path)
	f, err := utils.CreateFile(filename)
	if err != nil {
		log.Fatal(err.Error())
	}

	uiprogress.Start()
	chunkSize := int64(contentLength) / concurent
	counter := int64(0)
	var g errgroup.Group
	for i := int64(0); i < int64(contentLength); i += chunkSize {
		offset := i
		limit := i + chunkSize
		if limit > int64(contentLength) {
			limit = int64(contentLength)
		}

		g.Go(func() error {
			counter++
			return worker(counter, f, urlInput, offset, limit)
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	var urlInput string
	var concurrent int64
	flag.StringVar(&urlInput, "url", "", "url")
	flag.Int64Var(&concurrent, "concurrent", 1, "concurrent")
	flag.Parse()

	if urlInput == "" {
		log.Fatal("url cannot be empty")
	}

	_, err := url.ParseRequestURI(urlInput)
	if err != nil {
		log.Fatal(err.Error())
	}

	comot(urlInput, concurrent)
}

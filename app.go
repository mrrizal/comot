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
	"sync"

	"github.com/mrrizal/comot/utils"
)

func worker(wg *sync.WaitGroup, counterStream chan utils.CounterStream, workerID int, w io.WriterAt, urlInput string,
	off, limit int) error {
	defer wg.Done()

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

	_, err = io.Copy(utils.NewWriter(w, off, workerID, counterStream), resp.Body)
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

func comot(urlInput string, concurrent int) {
	// check is valid url
	resp, err := is_valid_url(urlInput)
	if err != nil {
		log.Fatal(err.Error())
	}

	// get content length
	contentLength := int(resp.ContentLength)
	if contentLength <= 0 {
		log.Fatal(errors.New("server sent invalid Content-Length Header"))
	}

	// create file
	filename := path.Base(resp.Request.URL.Path)
	f, err := utils.CreateFile(filename)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	counterStream := make(chan utils.CounterStream)
	limitOfssetData := utils.CountLimitOffset(contentLength, concurrent)
	bars := utils.SetupProgressBar(concurrent, limitOfssetData)

	// run worker
	go func() {
		defer close(counterStream)
		var wg sync.WaitGroup
		for key, value := range limitOfssetData {
			wg.Add(1)
			go worker(&wg, counterStream, key, f, urlInput, value.Offset, value.Limit)
		}
		wg.Wait()
	}()

	tracker := make(map[int]int)
	for i := range counterStream {
		key := i.ID - 1
		tracker[key] += i.Data
		bars[key].Set(tracker[key])
	}
}

func main() {
	var urlInput string
	var concurrent int
	flag.StringVar(&urlInput, "url", "", "url")
	flag.IntVar(&concurrent, "concurrent", 1, "concurrent")
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

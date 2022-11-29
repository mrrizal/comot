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
		return nil, err
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

func runWorker(counterStream chan utils.CounterStream, hasPartFile bool, partFileData map[int]utils.LimitOffsetData,
	filename, urlInput string, limitOffsetData map[int]utils.LimitOffsetData) error {
	f, err := utils.CreateFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// run worker
	defer close(counterStream)
	var wg sync.WaitGroup
	var data map[int]utils.LimitOffsetData
	if hasPartFile {
		data = partFileData
	} else {
		data = limitOffsetData
	}

	for key, value := range data {
		wg.Add(1)
		go worker(&wg, counterStream, key, f, urlInput, value.Offset, value.Limit)
	}
	wg.Wait()
	return nil
}

func fillingProgressBar(concurrent int, limitOffsetData map[int]utils.LimitOffsetData,
	counterStream chan utils.CounterStream, hasPartFile bool, partFileData map[int]utils.LimitOffsetData,
	tracker map[int]int) {
	// create progress bar
	bars := utils.SetupProgressBar(concurrent, limitOffsetData)

	// filling progress bar
	for i := range counterStream {
		if hasPartFile {
			if bars[i.ID].Current() == 0 {
				tracker[i.ID] += partFileData[i.ID].Offset - limitOffsetData[i.ID].Offset
				bars[i.ID].Set(partFileData[i.ID].Offset - limitOffsetData[i.ID].Offset)
			}
		}
		tracker[i.ID] += i.Data
		bars[i.ID].Set(tracker[i.ID])
	}
}

func comot(urlInput string, concurrent int) error {
	// check is valid url
	resp, err := is_valid_url(urlInput)
	if err != nil {
		return err
	}

	// get content length
	contentLength, err := utils.GetContentLenght(resp)
	if err != nil {
		return err
	}

	// create file
	filename := path.Base(resp.Request.URL.Path)

	// setup
	counterStream := make(chan utils.CounterStream)
	hasPartFile := utils.HasPartFile(filename, concurrent)
	limitOffsetData := utils.CountLimitOffset(contentLength, concurrent)
	var partFileData map[int]utils.LimitOffsetData
	if hasPartFile {
		partFileData, err = utils.ParsePartFile(filename, concurrent)
		if err != nil {
			return err
		}
	}
	tracker := utils.HandleCancelation(filename, limitOffsetData)

	// run worker
	go func() {
		err = runWorker(counterStream, hasPartFile, partFileData, filename, urlInput, limitOffsetData)
	}()

	if err != nil {
		return err
	}

	// filling progress bar
	fillingProgressBar(concurrent, limitOffsetData, counterStream, hasPartFile, partFileData, tracker)

	// it's handle to decide should i write part file or not.
	_, err = utils.IsDownloadComplete(filename, tracker, limitOffsetData)
	if err != nil {
		return err
	}
	return nil
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

	err = comot(urlInput, concurrent)
	if err != nil {
		log.Fatal(err.Error())
	}
}

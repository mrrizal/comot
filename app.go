package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
)

var client *fasthttp.Client

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

	readTimeout, _ := time.ParseDuration("500ms")
	writeTimeout, _ := time.ParseDuration("500ms")
	maxIdleConnDuration, _ := time.ParseDuration("1h")
	client = &fasthttp.Client{
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		MaxIdleConnDuration:           maxIdleConnDuration,
		NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
		DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this
		DisablePathNormalizing:        true,
		// increase DNS cache time to an hour instead of default minute
		Dial: (&fasthttp.TCPDialer{
			Concurrency:      4096,
			DNSCacheDuration: time.Hour,
		}).Dial,
	}

	comot(urlInput, concurrent)
}

func comot(urlInput string, concurent int) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(urlInput)
	req.Header.SetMethod(fasthttp.MethodHead)
	resp := fasthttp.AcquireResponse()
	err := client.Do(req, resp)
	fasthttp.ReleaseRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}
	contentLenght := resp.Header.ContentLength()
	fasthttp.ReleaseResponse(resp)

	if contentLenght <= 0 {
		log.Fatal("server sent invalid Content-Length Header")
	}

	chunkSize := contentLenght / concurent
	for i := 0; i < contentLenght; i += chunkSize {
		offset := i
		limit := i + chunkSize
		if limit > contentLenght {
			limit = contentLenght
		}
		fmt.Printf("offset %d, limit %d\n", offset, limit)
	}
	fmt.Println(contentLenght)
}

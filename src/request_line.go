package src

import (
	"errors"
	"fmt"
	"strings"
)

type RequestLine struct {
	method   string
	URI      string
	version  string
	download bool
}

const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)

const (
	DOWNLOAD = "DOWNLOADFILE"
	PRESENT  = "present"
)

func requestMethodImplemented(method string) bool {
	return (method == GET)
}

func httpVersionSupported(version string) bool {
	return (version == "HTTP/1.1")
}

func parseRequestLine(line string) (RequestLine, error) {
	split_line := strings.Split(line, " ")
	if len(split_line) != 3 {
		return RequestLine{}, fmt.Errorf("request line invalid: \"%s\", not enough arguments", line)
	}

	method := split_line[0]
	URI := split_line[1]
	version := split_line[2]
	download := false

	if strings.Contains(URI, "/"+DOWNLOAD+"/") {
		download = true
		URI = strings.TrimPrefix(URI, "/"+DOWNLOAD)
	}

	URI = strings.Replace(URI, "/"+DOWNLOAD, "", 1)

	request := RequestLine{
		method:   method,
		URI:      URI,
		version:  version,
		download: download,
	}

	if !httpVersionSupported(version) {
		return request, errors.New("VERSION NOT IMPLEMENTED")
	}

	if !requestMethodImplemented(method) {
		return request, errors.New("METHOD NOT IMPLEMENTED")
	}

	return request, nil
}

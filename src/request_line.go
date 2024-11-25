package src

import (
	"errors"
	"strings"
)

type RequestLine struct {
	method  string
	URI     string
	version string
}

const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
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
		return RequestLine{}, errors.New("REQUEST LINE INVALID: NOT ENOUGH ARGUMENTS")
	}

	method := split_line[0]
	URI := split_line[1]
	version := split_line[2]

	request := RequestLine{
		method:  method,
		URI:     URI,
		version: version,
	}

	if !httpVersionSupported(version) {
		return request, errors.New("VERSION NOT IMPLEMENTED")
	}

	if !requestMethodImplemented(method) {
		return request, errors.New("METHOD NOT IMPLEMENTED")
	}

	return request, nil
}

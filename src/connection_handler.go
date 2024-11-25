package src

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const CRLF = "\r\n"
const MAC_TCP_MSG_SIZE = 65537
const SUCCESS_CODE = 200

type ConnectionHandler struct {
	conn net.Conn
}

func (handler ConnectionHandler) Handle() {
	bytes := make([]byte, MAC_TCP_MSG_SIZE)
	length, _ := handler.conn.Read(bytes)
	request := string(bytes[:length])

	as_split_string := strings.Split(request, CRLF)
	request_line, err := parseRequestLine(as_split_string[0])

	if err != nil {
		if strings.HasSuffix(err.Error(), "NOT IMPLEMENTED") {
			HandleUnimplemented(handler.conn, request_line)
		} else {
			log.Fatal(err)
		}
	}

	switch request_line.method {
	case GET:
		result, err := handleGetRequest(request_line)
		if err != nil {
			log.Printf("Get request resulted in error: %s", err)
			handler.conn.Close()
		}

		handler.conn.Write([]byte(result.serialize()))

	default:
		HandleUnimplemented(handler.conn, request_line)
	}
}

func handleGetRequest(request_line RequestLine) (Response, error) {
	if !strings.HasPrefix(request_line.URI, "/") {
		return Response{}, fmt.Errorf("Request URI does not begin with \"/\"")
	}

	response_line, rl_err := createResponseLine("HTTP/1.1", SUCCESS_CODE, OK)

	if rl_err != nil {
		return Response{}, fmt.Errorf("Encountered error while constructing response line: %s\n", rl_err.Error())
	}

	content := strings.Join([]string{
		"<!DOCTYPE HTML>",
		"<html lang=\"en\">",
		"<head>",
		"<meta charset=\"utf-8\">",
		"<style type=\"text/css\">\n:root {\ncolor-scheme: light dark;\n}\n</style>",
		"<title>Basic HTTP Server in Golang</title>",
		"<body>Basic HTTP Server in Golang</body>",
	}, "\n") + "\n"

	headers := []string{
		"Server: go-http-server/0.0.1 Go/1.23",
		fmt.Sprintf("Date: %s", time.Now().Format(http.TimeFormat)),
		"Content-Type: text/html; charset=utf-8",
		fmt.Sprintf("Content-Length: %d", len(content)+4),
	}

	return Response{line: response_line, headers: headers, content: content}, nil
}

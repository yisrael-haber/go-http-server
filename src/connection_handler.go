package src

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

	content_of_requested_source, err := handlePathRequest(request_line)

	if err != nil {
		return Response{}, err
	}

	content := strings.Join([]string{
		"<!DOCTYPE HTML>",
		"<html lang=\"en\">",
		"<head>",
		"<meta charset=\"utf-8\">",
		"<style type=\"text/css\">\n:root {\ncolor-scheme: light dark;\n}\n</style>",
		"<title>Basic HTTP Server in Golang</title>",
		"<link rel='icon' href='resources/avhxh1b9r.webp' type='image/webp'/ >",
		"<body>",
		"<h1>Directory listing</h1>",
		content_of_requested_source,
		"</body>",
		"</html>",
	}, "\n") + "\n"

	headers := []string{
		"Server: go-http-server/0.0.1 Go/1.23",
		fmt.Sprintf("Date: %s", time.Now().Format(http.TimeFormat)),
		"Content-Type: text/html; charset=utf-8",
		fmt.Sprintf("Content-Length: %d", len(content)+4),
	}

	return Response{line: response_line, headers: headers, content: content}, nil
}

func handlePathRequest(line RequestLine) (string, error) {
	requested_path, err := confirmRequestedPath(line)

	if err != nil {
		return "", err
	}

	vec_that_will_be_returned := []string{"<hr>", "<ul>"}

	info, err := os.Stat(requested_path)

	if err != nil {
		return "", err
	}

	if info.IsDir() {
		read_dir, _ := os.ReadDir(requested_path)

		for _, entry := range read_dir {
			vec_that_will_be_returned = append(
				vec_that_will_be_returned,
				fmt.Sprintf("<li><a href=\"%s\">%s</a><li>", entry.Name(), entry.Name()),
			)
		}
	}

	vec_that_will_be_returned = append(vec_that_will_be_returned, "</ul>")
	vec_that_will_be_returned = append(vec_that_will_be_returned, "<hr>")

	return strings.Join(vec_that_will_be_returned, "\n"), nil
}

func confirmRequestedPath(line RequestLine) (string, error) {
	wd, err := os.Getwd()

	if err != nil {
		return "", errors.New("Cannot extract working directory, maybe should run server with elevated privileges.")
	}

	awd, err := filepath.Abs(wd)

	if err != nil {
		return "", errors.New("Could not extract absolute path of working directory. BIG PROBLEM.")
	}

	requested_abs_path, err := filepath.Abs("." + line.URI)

	if err != nil {
		fmt.Println(err)
		return "", errors.New("WOW MISTER YOU SEEM TO HAVE REQUESTED A PATH THAT SIMPLY DOES NOT EXIST.")
	}

	if strings.HasPrefix(awd, requested_abs_path) && (awd != requested_abs_path) {
		fmt.Println(awd)
		fmt.Println(requested_abs_path)
		return "", errors.New("Hey there bucko, note that you requested an illegal resource, maybe try running the server from a higher directory.")
	}

	return requested_abs_path, nil
}

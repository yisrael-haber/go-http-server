package src

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const CRLF = "\r\n"
const MAC_TCP_MSG_SIZE = 65537
const SUCCESS_CODE = 200
const MAX_FILE_SEND_SIZE = 1000 * 1000 * 100
const SERVER = "Server: go-http-server/0.0.1 Go/1.23"
const NBSP = "&nbsp;"

type ConnectionHandler struct {
	conn net.Conn
}

func AcceptAndHandleConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Connected to %s\n", conn.RemoteAddr().String())
		go func(c net.Conn) {
			ConnectionHandler{conn: c}.Handle()
			c.Close()
		}(conn)
	}
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
			log.Println(err)
			return
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

	ri, err := responseContentBuilder(request_line)

	if err != nil {
		return Response{}, err
	}

	content := ri.content
	headers := []string{}

	switch ri.IsDir {
	case true:
		headers = []string{
			SERVER,
			fmt.Sprintf("Date: %s", getDate()),
			"Content-Type: text/html; charset=utf-8",
			fmt.Sprintf("Content-Length: %d", len(content)),
		}

	case false:
		content_disposition := ""
		if request_line.download {
			content_disposition = "Content-Disposition: attachment"
		} else {
			content_disposition = "Content-Disposition: inline"
		}

		ct := http.DetectContentType([]byte(content[:min(len(content)-1, 512)]))
		headers = []string{
			SERVER,
			fmt.Sprintf("Date: %s", getDate()),
			fmt.Sprintf("Content-Type: %s; charset=utf-8", ct),
			fmt.Sprintf("Content-Length: %d", len(content)),
			content_disposition,
		}
	}

	return Response{line: response_line, headers: headers, content: content}, nil
}

func responseContentBuilder(line RequestLine) (ResponseInfo, error) {
	requested_path, err := confirmRequestedPath(line)

	ri := ResponseInfo{}

	if err != nil {
		return ri, err
	}

	info, err := os.Stat(requested_path)

	if err != nil {
		fmt.Printf("Requeste line %s resulted in error %s", line, err.Error())
		return ri, err
	}

	if info.IsDir() {
		lines := []string{
			"<!DOCTYPE HTML>",
			"<html lang=\"en\">",
			"<head>",
			"<meta charset=\"utf-8\">",
			"<style type=\"text/css\">\n:root {\ncolor-scheme: light dark;\n}\n</style>",
			"<title>Basic HTTP Server in Golang</title>",
			"</head>",
			"<body>",
			fmt.Sprintf("<h1>Directory Listing for \"%s\"</h1>", info.Name()),
			ri.content,
		}

		ri.IsDir = true
		read_dir, _ := os.ReadDir(requested_path)
		dirs := []string{}
		files := []string{}

		for _, entry := range read_dir {
			ei, _ := entry.Info()
			slash := ""
			if ei.IsDir() {
				slash = "/"
				space := strings.Repeat(NBSP, 2*(utf8.RuneCountInString(DOWNLOAD+" ")+1)+4)
				entry_line := fmt.Sprintf("<li>%s<a href=\"%s%s\">%s</a></li>", space, entry.Name(), slash, entry.Name())
				dirs = append(dirs, entry_line)
			} else {
				space := strings.Repeat(NBSP, 12)
				entry_line := fmt.Sprintf("<li><a href=\"%s/%s\"> download </a>%s<a href=\"%s%s\">%s</a></li>", DOWNLOAD, entry.Name(), space, entry.Name(), slash, entry.Name())
				files = append(files, entry_line)
			}
		}

		sort.Strings(dirs)
		sort.Strings(files)
		lines = append(lines, dirs...)
		lines = append(lines, files...)

		lines = append(lines, "</body>")
		lines = append(lines, "</html>")
		ri.content = strings.Join(lines, "\n")
	} else {
		ri.IsDir = false
		fb := make([]byte, MAX_FILE_SEND_SIZE)

		if info.Size() > MAX_FILE_SEND_SIZE {
			ri.content = fmt.Sprintf("Size of requested file %s, %d, was too large (larger than %d bytes)", info.Name(), info.Size(), MAX_FILE_SEND_SIZE)
		} else {
			file, err := os.Open(requested_path)
			if err != nil {
				ri.content = fmt.Sprintf("SERVER ERROR: could not open file %s due to error %s", info.Name(), err.Error())
			} else {
				num_bytes, err := file.Read(fb)
				if err != nil {
					ri.content = "SERVER ERROR: COULD NOT READ FILE"
				} else {
					ri.content = string(fb[:num_bytes])
				}
			}
		}
	}

	return ri, nil
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
		return "", errors.New("Hey there bucko, note that you requested an illegal resource, maybe try running the server from a higher directory.")
	}

	return requested_abs_path, nil
}

func getDate() string {
	return time.Now().Format(http.TimeFormat)
}

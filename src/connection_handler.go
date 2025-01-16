package src

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const CRLF = "\r\n"
const MAC_TCP_MSG_SIZE = 65537
const SUCCESS_CODE = 200
const MAX_FILE_SEND_SIZE = 1000 * 1000 * 100
const NBSP = "&nbsp;"

type ConnectionHandler struct {
	conn net.Conn
	wd   string
}

func (handler ConnectionHandler) Close() {
	handler.conn.Close()
}

func AcceptAndHandleConnections(listener net.Listener) {
	wd, err := os.Getwd()
	fmt.Println(wd)

	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			ConnectionHandler{conn: c, wd: wd}.Handle()
			c.Close()
		}(conn)
	}
}

type Header struct {
	description string
	value       string
}

var SERVER_HEADER = Header{description: "Server", value: "go-http-server/0.0.1 Go/1.23"}
var HTML_CONTENT_HEADER = Header{description: "Content-Type", value: "text/html; charset=utf-8"}

func (handler ConnectionHandler) send_rl(rl ResponseLine) {
	handler.conn.Write([]byte(rl.serialize() + CRLF))
}

func (header Header) serialize() string {
	return strings.Join([]string{header.description, header.value}, ": ")
}

func (handler ConnectionHandler) send_headers(headers []Header) {
	serialized_headers := []string{}
	for i := range len(headers) - 1 {
		serialized_headers = append(serialized_headers, headers[i].serialize())
	}
	serialized_headers = append(serialized_headers, CRLF)

	handler.conn.Write([]byte(strings.Join(serialized_headers, CRLF)))
}

func (handler ConnectionHandler) send_content(content string) {
	handler.conn.Write([]byte(content))
}

func (handler ConnectionHandler) Handle() {
	bytes := make([]byte, MAC_TCP_MSG_SIZE)
	length, _ := handler.conn.Read(bytes)
	request := string(bytes[:length])

	as_split_string := strings.Split(request, CRLF)
	request_line, err := parseRequestLine(as_split_string[0])
	log.Printf("Connected to %s, requested: %s\n", handler.conn.RemoteAddr().String(), request_line.URI)

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
		err := handleGetRequest(handler, request_line)
		if err != nil {
			if os.IsNotExist(err) {
				response_for_not_found(handler)
			} else {
				log.Printf("Get request resulted in error: %s", err)
			}

			handler.Close()
		}

	default:
		HandleUnimplemented(handler.conn, request_line)
	}
}

func handleGetRequest(handler ConnectionHandler, request_line RequestLine) error {
	if !strings.HasPrefix(request_line.URI, "/") {
		return fmt.Errorf("Request URI does not begin with \"/\"")
	}

	response_line, rl_err := createResponseLine("HTTP/1.1", SUCCESS_CODE, OK)
	if rl_err != nil {
		return fmt.Errorf("Encountered error while constructing response line: %s\n", rl_err.Error())
	}

	handler.send_rl(response_line)

	ri, err := responseContentBuilder(request_line, handler)

	if err != nil {
		return err
	}

	content := ri.content

	date_header := Header{description: "Date", value: getDate()}
	length_header := Header{description: "Content-Length", value: strconv.Itoa(len(content))}

	headers := []Header{
		SERVER_HEADER,
		date_header,
		length_header,
	}

	switch ri.IsDir {
	case true:
		headers = append(headers, HTML_CONTENT_HEADER)

	case false:
		content_disposition_header := Header{description: "Content-Disposition", value: ""}
		if request_line.download {
			content_disposition_header.value = "attachment"
		} else {
			content_disposition_header.value = "inline"
		}

		ct := http.DetectContentType([]byte(content))
		content_type_header := Header{description: "Content-Type", value: fmt.Sprintf("%s; charset=utf-8", ct)}

		headers = append(headers, content_disposition_header)
		headers = append(headers, content_type_header)
	}

	handler.send_headers(headers)
	handler.send_content(content)

	return nil
}

func responseContentBuilder(line RequestLine, handler ConnectionHandler) (ResponseInfo, error) {
	requested_path, err := confirmRequestedPath(line, handler)
	ri := ResponseInfo{}

	if err != nil {
		return ri, err
	}

	index_candidate_path := filepath.Join(requested_path, "index.html")
	if info, err := os.Stat(index_candidate_path); err == nil {
		build_content_from_file(index_candidate_path, &ri, info)
	} else {
		info, err := os.Stat(requested_path)

		if err != nil {
			return ri, err
		}

		if info.IsDir() {
			build_content_from_directory(requested_path, &ri, handler)
		} else {
			build_content_from_file(requested_path, &ri, info)
		}
	}

	return ri, nil
}

func build_content_from_file(fp string, ri *ResponseInfo, info fs.FileInfo) {
	log.Printf("Building content from %s", fp)

	ri.IsDir = false
	fb := make([]byte, MAX_FILE_SEND_SIZE)

	if info.Size() > MAX_FILE_SEND_SIZE {
		ri.content = fmt.Sprintf("Size of requested file %s, %d, was too large (larger than %d bytes)", info.Name(), info.Size(), MAX_FILE_SEND_SIZE)
	} else {
		file, err := os.Open(fp)
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

func build_content_from_directory(fp string, ri *ResponseInfo, handler ConnectionHandler) {
	_, dirname := filepath.Split(fp)
	lines := []string{
		"<!DOCTYPE HTML>",
		"<html lang=\"en\">",
		"<head>",
		"<meta charset=\"utf-8\">",
		"<style type=\"text/css\">\n:root {\ncolor-scheme: light dark;\n}\n</style>",
		"<title>Basic HTTP Server in Golang</title>",
		"</head>",
		"<body>",
		fmt.Sprintf("<h1>Directory Listing for %s</h1>", dirname),
		ri.content,
	}

	ri.IsDir = true
	read_dir, _ := os.ReadDir(fp)

	dirs := []string{}
	files := []string{}

	for _, entry := range read_dir {
		ei, _ := entry.Info()
		slash := ""

		rel_path, err := filepath.Rel(handler.wd, filepath.Join(fp, ei.Name()))
		if err != nil {
			log.Printf("Tried to make realtive path from \"%s\" and \"%s\"", handler.wd, fp)
		}

		if ei.IsDir() {
			slash = "/"
			space := strings.Repeat(NBSP, 2*(utf8.RuneCountInString(DOWNLOAD+" ")+1)+4)
			entry_line := fmt.Sprintf("<li>%s<a href=\"%s\">%s</a></li>", space, slash+rel_path+slash, entry.Name())
			dirs = append(dirs, entry_line)
		} else {
			space := strings.Repeat(NBSP, 12)
			entry_line := fmt.Sprintf(
				"<li><a href=\"%s/%s\"> download </a>%s<a href=\"%s\">%s</a></li>",
				DOWNLOAD, rel_path, space, slash+rel_path, entry.Name())
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
}

func confirmRequestedPath(line RequestLine, handler ConnectionHandler) (string, error) {
	requested_abs_path, err := filepath.Abs("." + line.URI)

	if err != nil {
		fmt.Println(err)
		return "", errors.New("WOW MISTER YOU SEEM TO HAVE REQUESTED A PATH THAT SIMPLY DOES NOT EXIST.")
	}

	if strings.HasPrefix(handler.wd, requested_abs_path) && (handler.wd != requested_abs_path) {
		return "", errors.New("Hey there bucko, note that you requested an illegal resource, maybe try running the server from a higher directory.")
	}

	return requested_abs_path, nil
}

func getDate() string {
	return time.Now().Format(http.TimeFormat)
}

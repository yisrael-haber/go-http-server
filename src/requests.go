package src

import "strings"

type Request struct {
	line    RequestLine
	headers []string
}

type Response struct {
	line    ResponseLine
	headers []string
	content string
}

func (response Response) serialize() string {
	return response.line.serialize() + CRLF + strings.Join(response.headers, CRLF) + CRLF + CRLF + response.content
}

package src

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const NOT_IMPLEMENTED_CODE = 501

func HandleUnimplemented(conn net.Conn, request_line RequestLine) {
	return_str := fmt.Sprintf("%s %d Not Implemented%%s", request_line.version, NOT_IMPLEMENTED_CODE, CRLF, CRLF)
	conn.Write([]byte(return_str))
}

func BindPort(port int) net.Listener {
	bind_addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", bind_addr)

	if err != nil {
		log.Fatal(err)
	}

	return l
}

func DisplayHelp() {
	fmt.Println("Usage: ./server [help|--port=[PORT] --loc=[LOCATION]]")
	fmt.Println("\tWhere PORT is a valid TCP port to serve on, 9998 by default.")
	fmt.Println("\tWhere LOCATION is a valid TCP port to serve on, \".\" by default.")
}

func ExtractArgs(args []string) (int, string, error) {
	port := 10101
	loc, _ := filepath.Abs(".")

	for _, arg := range args {
		if strings.HasPrefix(arg, "--port=") {
			right_side := strings.SplitN(arg, "=", 2)
			suggested_port, err := strconv.Atoi(right_side[1])
			fmt.Printf("Suggested port is %d", suggested_port)

			if err != nil {
				return port, loc, errors.New(fmt.Sprintf("Provided port %s illegal\n", right_side[1]))
			}

			if suggested_port < 1000 || suggested_port > 40000 {
				return port, loc, errors.New(fmt.Sprintf("Provided port %s not in valid range. Requires 1000<PORT<40000.\n", right_side[1]))
			}

			port = suggested_port
		}

		if strings.HasPrefix(arg, "--loc=") {
			right_side := strings.SplitN(arg, "=", 2)

			_, err := os.Stat(right_side[1])

			if err != nil {
				return port, loc, errors.New(fmt.Sprintf("While looking for path %s, encountered error:%s\n", right_side[1], err.Error()))
			}

			loc, _ = filepath.Abs(right_side[1])
		}
	}

	return port, loc, nil
}

func response_for_not_found(handler ConnectionHandler) {
	rl := ResponseLine{version: "HTTP/1.1", status_code: 404, message: "File not found"}
	handler.send_rl(rl)

	content := `<!DOCTYPE HTML>
<html lang="en">
    <head>
        <meta charset="utf-8">
        <title>Error response</title>
    </head>
    <body>
        <h1>Error response</h1>
        <p>Error code: 404</p>
        <p>Message: File not found.</p>
        <p>Error code explanation: 404 - Nothing matches the given URI.</p>
    </body>
</html>
`

	date_header := Header{description: "Date", value: getDate()}
	length_header := Header{description: "Content-Length", value: string(len(content))}

	headers := []Header{
		SERVER_HEADER,
		date_header,
		length_header,
		HTML_CONTENT_HEADER,
		Header{description: "Connection", value: "close"},
	}

	handler.send_headers(headers)
	handler.send_content(content)
}

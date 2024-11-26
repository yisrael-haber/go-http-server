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

func DisplayHelp() {
	fmt.Println("Usage: ./server [help|--port==[PORT] --loc==[LOCATION]]")
	fmt.Println("\tWhere PORT is a valid TCP port to serve on, 9998 by default.")
	fmt.Println("\tWhere LOCATION is a valid TCP port to serve on, \".\" by default.")
}

func ExtractArgs(args []string) (int, string, error) {
	port := 10101
	loc, _ := filepath.Abs(".")

	for _, arg := range args {
		fmt.Println(arg)
		fmt.Println(strings.HasPrefix(arg, "--port="))
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

package src

import (
	"fmt"
)

const OK = "OK"

type ResponseLine struct {
	version     string
	status_code int
	message     string
}

func createResponseLine(version string, status_code int, message string) (ResponseLine, error) {

	if status_code < 0 || status_code > 600 {
		return ResponseLine{}, fmt.Errorf("Cannot create response line with status code %d", status_code)
	}

	if message != OK {
		return ResponseLine{}, fmt.Errorf("Cannot create response line with message %s", message)
	}

	if version != "HTTP/1.1" {
		return ResponseLine{}, fmt.Errorf("Cannot create response line with version %s", version)
	}

	return ResponseLine{
		version:     version,
		status_code: status_code,
		message:     message,
	}, nil
}

func (rl ResponseLine) serialize() string {
	return fmt.Sprintf("%s %d %s", rl.version, rl.status_code, rl.message)
}

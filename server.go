package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const ip = "127.0.0.1"
const port = "6969"

func main() {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		fmt.Printf("Failed to connect at port %s \n", port)
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection", err.Error())
		os.Exit(1)
	}

	// Write a basic HTTP response to the client with status 200 OK
	// \r\n\r\n indicates the end of the HTTP headers
	// conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	// req takes the request data from the client
	// it is a byte slice with a size of 4096 bytes
	req := make([]byte, 4096)

	conn.Read(req)

	// Example: Suppose req contains "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	// parts: a slice of strings split by "\r\n", e.g. ["GET / HTTP/1.1", "Host: localhost", "", ""]
	// reqLinePart: a slice of strings split by spaces from the first line, e.g. ["GET", "/", "HTTP/1.1"]
	// The first part is the request line, the second part is the request URI, and the third part is the HTTP version
	parts := strings.Split(string(req), "\r\n")
	reqLinePart := strings.Split(parts[0], " ")

	// Respond with 200 OK for "/" path, 404 Not Found otherwise.
	if reqLinePart[1] == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

		// /echo/{content}
	} else if strings.HasPrefix(reqLinePart[1], "/echo") {
		// if reqLinePart[1] is /echo/auco then uriParts is ["", "echo", "auco"]
		uriParts := strings.Split(reqLinePart[1], "/")

		if len(uriParts) != 3 {
			// Handle invalid echo requests (too few or too many parts)
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		} else {
			// Valid echo request: exactly 3 parts ["", "echo", "content"]
			str := uriParts[2]
			strLen := len(str)

			// Content-Type: text/plain indicates plain text response
			// Content-Length: strLen specifies the length of the response body
			// The response body is the string extracted from the URI
			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", strLen, str)
			conn.Write([]byte(response))
		}

	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		conn.Close()
	}
}

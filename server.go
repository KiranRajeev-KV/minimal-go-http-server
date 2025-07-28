package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const ip = "127.0.0.1"
const port = "6969"

// Define the temporary directory
const tempDirectory = "./files"

// Log file path
const logFilePath = "./server.log"

// Single logger function with log level parameter
func log(level, format string, args ...any) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)

	// Print to console
	fmt.Print(logEntry)

	// Append to log file
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(logEntry)
	if err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}

type HTTPRequest struct {
	Headers map[string]string
	Url     string
	Method  string
	Body    []byte
}

func listenReq(conn net.Conn) {
	defer conn.Close()

	// rawReq takes the request data from the client
	// it is a byte slice with a size of 4096 bytes
	rawReq := make([]byte, 4096)

	bytesRead, err := conn.Read(rawReq)
	if err != nil {
		log("ERROR", "Failed to read request from %s: %v", conn.RemoteAddr(), err)
		return
	}

	log("INFO", "Read %d bytes from %s", bytesRead, conn.RemoteAddr())

	// Example: Suppose rawReq contains "POST /echo/hello HTTP/1.1\r\nHost: localhost\r\nUser-Agent: curl/8.0.1\r\nContent-Length: 5\r\n\r\nworld"
	// parts: split by "\r\n\r\n" to separate headers from body, e.g. ["POST /echo/hello HTTP/1.1\r\nHost: localhost\r\nUser-Agent: curl/8.0.1\r\nContent-Length: 5", "world"]
	// metaParts: split headers section by "\r\n", e.g. ["POST /echo/hello HTTP/1.1", "Host: localhost", "User-Agent: curl/8.0.1", "Content-Length: 5"]
	// reqLinePart: split the first line by spaces, e.g. ["POST", "/echo/hello", "HTTP/1.1"]
	// The body is "world"
	parts := strings.Split(string(rawReq), "\r\n\r\n")
	metaParts := strings.Split(parts[0], "\r\n")
	reqLinePart := strings.Split(metaParts[0], " ")

	headers := make(map[string]string)

	// Parse headers from metaParts[1:] and store in headers map
	// Each header line is "Key: Value"
	// Example: "Host: localhost" will be stored as headers["Host"] = "localhost"
	for i := 1; i < len(metaParts); i++ {
		headerParts := strings.Split(metaParts[i], ":")
		if len(headerParts) >= 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(strings.Join(headerParts[1:], ":"))
			headers[key] = value
		}
	}

	// Parse Content-Length header first
	contentLength := 0
	contentLengthStr, exists := headers["Content-Length"]
	if exists {
		parsedLength, err := strconv.Atoi(contentLengthStr)
		if err == nil {
			contentLength = parsedLength
		}
	}

	// Get body if present, respecting Content-Length
	var body []byte
	if len(parts) > 1 && contentLength > 0 {
		// Ensure we don't read beyond the available data
		if contentLength <= len(parts[1]) {
			body = []byte(parts[1][:contentLength])
		} else {
			// Content-Length is larger than available data (partial read)
			body = []byte(parts[1])
		}
	}

	request := HTTPRequest{
		Url:     reqLinePart[1],
		Headers: headers,
		Method:  reqLinePart[0],
		Body:    body,
	}

	// Log the incoming request
	log("REQUEST", "%s %s from %s", request.Method, request.Url, conn.RemoteAddr().String())
	if len(request.Body) > 0 {
		log("INFO", "Request body: %d bytes", len(request.Body))
	}

	// Respond with 200 OK for "/" path, 404 Not Found otherwise.
	if request.Url == "/" {
		log("INFO", "Serving root path")
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		log("RESPONSE", "200 for %s", request.Url)

		// /echo/{content}
	} else if strings.HasPrefix(request.Url, "/echo") {
		// if request.Url is /echo/auco then uriParts is ["", "echo", "auco"]
		uriParts := strings.Split(request.Url, "/")

		if len(uriParts) != 3 {
			// Handle invalid echo requests (too few or too many parts)
			log("ERROR", "Invalid echo request format: %s", request.Url)
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			log("RESPONSE", "404 for %s", request.Url)
		} else {
			// Valid echo request: exactly 3 parts ["", "echo", "content"]
			str := uriParts[2]
			strLen := len(str)

			log("INFO", "Echoing content: '%s'", str)
			// Content-Type: text/plain indicates plain text response
			// Content-Length: strLen specifies the length of the response body
			// The response body is the string extracted from the URI
			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", strLen, str)
			conn.Write([]byte(response))
			log("RESPONSE", "200 for %s", request.Url)
		}
		// /user-agent
	} else if strings.HasPrefix(request.Url, "/user-agent") {
		usrAg := request.Headers["User-Agent"]

		if usrAg == "" {
			log("ERROR", "User-Agent header missing")
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			log("RESPONSE", "400 for %s", request.Url)
		} else {
			log("INFO", "User-Agent: %s", usrAg)

			usrAgLen := len(usrAg)
			response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", usrAgLen, usrAg)

			conn.Write([]byte(response))
			log("RESPONSE", "200 for %s", request.Url)
		}
		// /get-file or /file for GET/POST
	} else if strings.HasPrefix(request.Url, "/file") {
		log("INFO", "Handling file request: %s %s", request.Method, request.Url)
		// if request.Url is /file/somefile.txt then uriParts is ["", "file", "somefile.txt"]
		uriParts := strings.Split(request.Url, "/")

		if len(uriParts) != 3 {
			log("ERROR", "Invalid file request format: %s", request.Url)
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			log("RESPONSE", "404 for %s", request.Url)
		} else {
			path := uriParts[2]
			filePath := fmt.Sprintf("%s/%s", tempDirectory, path)

			switch request.Method {
			case "GET":
				log("INFO", "GET file: %s", filePath)

				if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
					log("ERROR", "File not found: %s", filePath)

					conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
					log("RESPONSE", "404 for %s", request.Url)
				} else {
					// File exists, read and serve it
					file, err := os.Open(filePath)
					if err != nil {
						log("ERROR", "Error opening file %s: %v", filePath, err)

						conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
						log("RESPONSE", "500 for %s", request.Url)
						return
					}
					defer file.Close()

					content, err := io.ReadAll(file)
					if err != nil {
						log("ERROR", "Error reading file %s: %v", filePath, err)

						conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
						log("RESPONSE", "500 for %s", request.Url)
						return
					}

					log("INFO", "Serving file %s (%d bytes)", filePath, len(content))

					response := fmt.Sprintf(
						"HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
						len(content), content,
					)
					conn.Write([]byte(response))
					log("RESPONSE", "200 for %s", request.Url)
				}
			case "POST":
				log("INFO", "POST file: %s (%d bytes)", filePath, len(request.Body))
				
				// Write request.Body to the file
				file, err := os.Create(filePath)
				if err != nil {
					log("ERROR", "Error creating file %s: %v", filePath, err)

					conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
					log("RESPONSE", "500 for %s", request.Url)
					return
				}
				defer file.Close()

				_, err = file.Write(request.Body)
				if err != nil {
					log("ERROR", "Error writing to file %s: %v", filePath, err)

					conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
					log("RESPONSE", "500 for %s", request.Url)
					return
				}

				log("INFO", "Successfully created file: %s", filePath)
				
				conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
				log("RESPONSE", "201 for %s", request.Url)
			default:
				log("ERROR", "Method not allowed: %s", request.Method)
				conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
				log("RESPONSE", "405 for %s", request.Url)
			}
		}
	} else {
		log("ERROR", "Route not found: %s %s", request.Method, request.Url)
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		log("RESPONSE", "404 for %s", request.Url)
	}
}

func main() {
	log("INFO", "Starting HTTP server on %s:%s", ip, port)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log("ERROR", "Failed to start server: %v", err)
		os.Exit(1)
	}

	log("INFO", "Server successfully started and listening for connections")

	// Accept connections in a loop to handle multiple clients
	for {
		conn, err := l.Accept()
		if err != nil {
			log("ERROR", "Error accepting connection: %v", err)
			continue // Continue to accept other connections
		}

		log("INFO", "New connection from %s", conn.RemoteAddr())
		// Handle each connection concurrently using goroutines
		go listenReq(conn)
	}
}

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

type HTTPResponse struct {
    Headers map[string]string
    Code    int
    Body    []byte
}

const StatusOK = 200
const StatusCreated = 201
const StatusNotFound = 404
const StatusInternalServerError = 500
const StatusMethodNotAllowed = 405

func StatusText(code int) string {
    switch code {
    case StatusOK:
        return "OK"
    case StatusCreated:
        return "Created"
    case StatusNotFound:
        return "Not Found"
    case StatusInternalServerError:
		return "Internal Server Error"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	}

    return ""
}

func (response HTTPResponse) Write() []byte {
	// standard HTTP response format of "HTTP/1.1 {code} {status}\r\n{headers}\r\n{body}"
    str := fmt.Sprintf("HTTP/1.1 %d %s\r\n", response.Code, StatusText(response.Code))

	// add headers to the response
    for header, value := range response.Headers {
        str += fmt.Sprintf("%s: %s\r\n", header, value)
    }
	
	// add Content-Length header if body is present
    if len(response.Body) > 0 {
        str += fmt.Sprintf("Content-Length: %d\r\n", len(response.Body))
    }
	// end of headers section
    str += "\r\n"

	// append body if present
    if len(response.Body) > 0 {
        str += string(response.Body)
    }
    return []byte(str)
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
		response := HTTPResponse{
			Code: StatusOK,
			Headers: headers,
		}
		log("INFO", "Serving root path")
		conn.Write(response.Write())
		log("RESPONSE", "200 for %s", request.Url)

		// /echo/{content}
	} else if strings.HasPrefix(request.Url, "/echo") {
		// if request.Url is /echo/auco then uriParts is ["", "echo", "auco"]
		uriParts := strings.Split(request.Url, "/")

		if len(uriParts) != 3 {
			// Handle invalid echo requests (too few or too many parts)
			response := HTTPResponse{
				Code: StatusNotFound,
				Headers: headers,
			}
			log("ERROR", "Invalid echo request format: %s", request.Url)
			conn.Write(response.Write())
			log("RESPONSE", "404 for %s", request.Url)
		} else {
			// Valid echo request: exactly 3 parts ["", "echo", "content"]
			str := uriParts[2]
			strLen := len(str)

			log("INFO", "Echoing content: '%s'", str)
			// Content-Type: text/plain indicates plain text response
			// Content-Length: strLen specifies the length of the response body
			// The response body is the string extracted from the URI
			response := HTTPResponse{
				Code: StatusOK,
				Headers: map[string]string{
					"Content-Type":   "text/plain",
					"Content-Length": fmt.Sprintf("%d", strLen),
				},
				Body: []byte(str),
			}
			conn.Write(response.Write())
			log("RESPONSE", "200 for %s", request.Url)
		}
		// /user-agent
	} else if strings.HasPrefix(request.Url, "/user-agent") {
		usrAg := request.Headers["User-Agent"]

		if usrAg == "" {
			// User-Agent header is missing, respond with 400 Bad Request
			log("ERROR", "User-Agent header missing")
			response := HTTPResponse{
				Code: StatusNotFound,
				Headers: headers,
			}
			conn.Write(response.Write())
			log("RESPONSE", "400 for %s", request.Url)
		} else {
			log("INFO", "User-Agent: %s", usrAg)

			usrAgLen := len(usrAg)
			response := HTTPResponse{
				Code: StatusOK,
				Headers: map[string]string{
					"Content-Length": fmt.Sprintf("%d", usrAgLen),
				},
				Body: []byte(usrAg),
			}
			conn.Write(response.Write())
			log("RESPONSE", "200 for %s", request.Url)
		}
		// /get-file or /file for GET/POST
	} else if strings.HasPrefix(request.Url, "/file") {
		log("INFO", "Handling file request: %s %s", request.Method, request.Url)
		// if request.Url is /file/somefile.txt then uriParts is ["", "file", "somefile.txt"]
		uriParts := strings.Split(request.Url, "/")

		if len(uriParts) != 3 {
			log("ERROR", "Invalid file request format: %s", request.Url)
			response := HTTPResponse{
				Code: StatusNotFound,
				Headers: headers,
			}
			conn.Write(response.Write())
			log("RESPONSE", "404 for %s", request.Url)
		} else {
			path := uriParts[2]
			filePath := fmt.Sprintf("%s/%s", tempDirectory, path)

			switch request.Method {
			case "GET":
				log("INFO", "GET file: %s", filePath)

				if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
					log("ERROR", "File not found: %s", filePath)

					response := HTTPResponse{
						Code: StatusNotFound,
						Headers: headers,
					}
					conn.Write(response.Write())
					log("RESPONSE", "404 for %s", request.Url)
				} else {
					// File exists, read and serve it
					file, err := os.Open(filePath)
					if err != nil {
						log("ERROR", "Error opening file %s: %v", filePath, err)

						response := HTTPResponse{
							Code: StatusInternalServerError,
							Headers: headers,
						}
						conn.Write(response.Write())
						log("RESPONSE", "500 for %s", request.Url)
						return
					}
					defer file.Close()

					content, err := io.ReadAll(file)
					if err != nil {
						log("ERROR", "Error reading file %s: %v", filePath, err)

						response := HTTPResponse{
							Code: StatusInternalServerError,
							Headers: headers,
						}
						conn.Write(response.Write())
						log("RESPONSE", "500 for %s", request.Url)
						return
					}

					log("INFO", "Serving file %s (%d bytes)", filePath, len(content))

					response := HTTPResponse{
						Code: StatusOK,
						Headers: map[string]string{
							"Content-Type":   "application/octet-stream",
							"Content-Length": fmt.Sprintf("%d", len(content)),
						},
						Body: content,
					}
					conn.Write(response.Write())
					log("RESPONSE", "200 for %s", request.Url)
				}
			case "POST":
				log("INFO", "POST file: %s (%d bytes)", filePath, len(request.Body))
				
				// Write request.Body to the file
				file, err := os.Create(filePath)
				if err != nil {
					log("ERROR", "Error creating file %s: %v", filePath, err)

					response := HTTPResponse{
						Code: StatusInternalServerError,
						Headers: headers,
					}
					conn.Write(response.Write())
					log("RESPONSE", "500 for %s", request.Url)
					return
				}
				defer file.Close()

				_, err = file.Write(request.Body)
				if err != nil {
					log("ERROR", "Error writing to file %s: %v", filePath, err)

					response := HTTPResponse{
						Code: StatusInternalServerError,
						Headers: headers,
					}
					conn.Write(response.Write())
					log("RESPONSE", "500 for %s", request.Url)
					return
				}

				log("INFO", "Successfully created file: %s", filePath)

				response := HTTPResponse{
					Code: StatusCreated,
					Headers: headers,
				}
				conn.Write(response.Write())
				log("RESPONSE", "201 for %s", request.Url)
			default:
				log("ERROR", "Method not allowed: %s", request.Method)
				response := HTTPResponse{
					Code: StatusMethodNotAllowed,
					Headers: headers,
				}
				conn.Write(response.Write())
				log("RESPONSE", "405 for %s", request.Url)
			}
		}
	} else {
		log("ERROR", "Route not found: %s %s", request.Method, request.Url)
		response := HTTPResponse{
			Code: StatusNotFound,
			Headers: headers,
		}
		conn.Write(response.Write())
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

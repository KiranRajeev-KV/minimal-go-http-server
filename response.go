package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"
)

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

func (response HTTPResponse) Write(request HTTPRequest) []byte {
	// Handle gzip encoding if requested
	if encodingsStr, ok := request.Headers["Accept-Encoding"]; ok {
		// Example: if encodingsStr = "gzip, deflate", then encodings will be ["gzip", "deflate"]
		encodings := strings.SplitSeq(encodingsStr, ", ")
		for encoding := range encodings {
			if encoding == "gzip" {
				// Create a buffer to hold the compressed content
				var encodedContent bytes.Buffer

				// Create a gzip writer to compress the response body
				gz := gzip.NewWriter(&encodedContent)

				if _, err := gz.Write(response.Body); err != nil {
					log("ERROR", "Failed to compress response body: %v", err)
				} else {
					gz.Close()
					response.Headers["Content-Encoding"] = encoding
					response.Body = encodedContent.Bytes()
				}
				break
			}
		}
	}

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

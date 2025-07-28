package main

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
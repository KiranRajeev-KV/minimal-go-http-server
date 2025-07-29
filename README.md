# Minimal Go HTTP Server

A minimal HTTP server implemented in Go using the `net` package. It supports basic routing, request parsing, and file I/O — all without third-party libraries.
> **Reference**: This project was built following the concepts from [this blog post](https://www.krayorn.com/posts/http-server-go/).
---

## Features

- Basic routing and request parsing
- Logging to file and stdout
- Echoing URI content
- Reflecting back `User-Agent` header
- File upload/download (via POST/GET)
- Custom HTTP response builder with support for gzip

---

## Project Structure

| File         | Purpose                                 |
|--------------|-----------------------------------------|
| `server.go`  | Main server loop and route handling     |
| `logger.go`  | Logging logic with timestamps and levels|
| `types.go`   | Defines `HTTPRequest` and `HTTPResponse`|
| `response.go`| Custom response formatting (gzip-ready) |
| `./files/`   | Stores uploaded/downloaded files        |
| `server.log` | Log file created at runtime             |

---

## Routing Table

| Route               | Method | Description                                              |
|---------------------|--------|----------------------------------------------------------|
| `/`                 | GET    | Responds with `200 OK`.                                  |
| `/echo/{message}`   | GET    | Returns `{message}` as plain text.                       |
| `/user-agent`       | GET    | Echoes the `User-Agent` header from the request.         |
| `/file/{filename}`  | GET    | Serves file from `./files/{filename}`.                   |
| `/file/{filename}`  | POST   | Saves uploaded content to `./files/{filename}`.          |
| `/logs`             | GET    | Returns the contents of `server.log` as plain text.      |

---

## How to Run

```bash
go run .
```

The server listens on `127.0.0.1:6969`.

---

## Example Usage

#### `/` Root Route

```bash
curl http://localhost:6969/
```

#### `/echo/{message}`

```bash
curl http://localhost:6969/echo/hello
```

#### `/user-agent`

```bash
curl http://localhost:6969/user-agent
```

#### `/file/{filename}` — Upload

```bash
curl -X POST --data-binary @example.txt http://localhost:6969/file/example.txt
```

#### `/file/{filename}` — Download

```bash
curl http://localhost:6969/file/example.txt
```
#### `/logs`

```bash
curl http://localhost:6969/logs
```

---


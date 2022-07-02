package sitanhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"
)

func DebugCheck(e error) {
	if e != nil {
		log.Println(e)
	}
}

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	DebugCheck(err)
	for {
		conn, err := l.Accept()
		DebugCheck(err)
		go s.HandleConnection(conn)

	}
	// Hint: call HandleConnection
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	// Hint: use the other methods below
	log.Println("have new request")
	br := bufio.NewReader(conn)
	for {
		// Set timeout
		log.Println("set timeout")
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		DebugCheck(err)
		req, bytesReceived, err := ReadRequest(br)
		if err != nil {
			var res Response

			log.Println("request erro")
			if err == io.EOF {
				conn.Close()
				break
			} else if _, ok := err.(*RequestError); ok {
				log.Println("RequestError")

				res.HandleBadRequest()
			} else if nErr, ok := err.(net.Error); ok {
				if nErr.Timeout() {
					if bytesReceived {
						res.HandleBadRequest()
					} else {
						conn.Close()
						break
					}
				}
			}
			// Try to read next request

			// FilePath is the local path to the file to serve.
			// It could be "", which means there is no file to serve.

			// Handle EOF

			// Handle timeout

			// Handle bad request
			res.Write(conn)
			// Close conn if requested
			if res.Header["Connection"] == "close" {
				conn.Close()
				break
			}
		} else {
			res := s.HandleGoodRequest(req)
			res.Write(conn)
			// Close conn if requested
			if res.Header["Connection"] == "close" || res.StatusCode == 400 {
				conn.Close()
				break
			}
		}
	}
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) *Response {
	var res Response
	res.Proto = "HTTP/1.1"
	res.Header = make(map[string]string)

	rerr, uri := req.ParseURI(s.DocRoot)
	if rerr != nil {
		res.HandleBadRequest()
		return &res

	}

	fileinfo, err := os.Stat(uri)
	if errors.Is(err, fs.ErrNotExist) || fileinfo.IsDir() {
		// if err != nil {
		res.HandleNotFound(req)
		return &res
	}
	res.HandleOK(req, uri)
	return &res
	// Hint: use the other methods below
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	res.FilePath = path
	res.StatusCode = 200
	fileinfo, _ := os.Stat(path)
	ext := filepath.Ext(path)
	mediaType := mime.TypeByExtension(ext)
	res.Header["Date"] = FormatTime(time.Now())

	res.Header[CanonicalHeaderKey("Content-Type")] = mediaType
	res.Header[CanonicalHeaderKey("Content-Length")] = fmt.Sprintf("%d", fileinfo.Size())
	res.Header[CanonicalHeaderKey("Last-Modified")] = FormatTime(fileinfo.ModTime())
	if req.Close {
		res.Header["Connection"] = "close"
	}

}

func (req *Request) ParseURI(root string) (error, string) {
	url := req.URL
	lenURL := len(url)
	if lenURL == 0 {
		return &RequestError{
			info:       "no url",
			statusCode: 400,
		}, ""
	}
	if url[0] != '/' {

		return &RequestError{
			info:       " URI need to start with '/'" + url,
			statusCode: 400,
		}, ""
	}
	if url[lenURL-1] == '/' {
		url += "index.html"
	}
	url = url[1:]
	root, _ = filepath.Abs(root)
	url = path.Clean(url)
	url = filepath.Join(root, url)
	url = path.Clean(url)

	rel, _ := filepath.Rel(root, url)
	if rel[0:2] == ".." {

		return &RequestError{
			info:       "root attacker!",
			statusCode: 400,
		}, ""
	}

	return nil, url
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {
	res.Proto = "HTTP/1.1"
	res.Header = make(map[string]string)
	res.StatusCode = 400
	res.Header["Date"] = FormatTime(time.Now())

	res.Header["Connection"] = "close"

}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	res.Header = make(map[string]string)
	if req.Close {
		res.Header["Connection"] = "close"
	}
	res.Header["Date"] = FormatTime(time.Now())

	res.StatusCode = 404
}

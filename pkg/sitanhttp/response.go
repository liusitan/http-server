package sitanhttp

import (
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"sort"
)

type Response struct {
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"

	// Header stores all headers to write to the response.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// Write writes the res to the w.
func (res *Response) Write(w io.Writer) error {
	fmt.Println("writing response")
	if err := res.WriteStatusLine(w); err != nil {
		return err
	}
	if err := res.WriteSortedHeaders(w); err != nil {
		return err
	}
	if res.StatusCode == 200 {
		if err := res.WriteBody(w); err != nil {
			return err
		}
	}

	return nil
}

type ResponseError struct {
	info string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("status code: %s", e.info)
}

// WriteStatusLine writes the status line of res to w, including the ending "\r\n".
// For example, it could write "HTTP/1.1 200 OK\r\n".
func (res *Response) WriteStatusLine(w io.Writer) error {
	var err error
	bw := bufio.NewWriter(w)
	wtp := textproto.NewWriter(bw)
	var msg string
	if res.StatusCode == 200 {
		msg = "OK"
	} else if res.StatusCode == 400 {
		msg = "Bad Request"
	} else if res.StatusCode == 404 {
		msg = "Not Found"
	} else {
		return &ResponseError{
			info: "wrong status code",
		}
	}
	fmt.Printf("%s %d %s\n", res.Proto, res.StatusCode, msg)
	if err = wtp.PrintfLine("%s %d %s", res.Proto, res.StatusCode, msg); err != nil {
		return err
	}

	err = bw.Flush()
	return err
}

// WriteSortedHeaders writes the headers of res to w, including the ending "\r\n".
// For example, it could write "Connection: close\r\nDate: foobar\r\n\r\n".
// For HTTP, there is no need to write headers in any particular order.
// sitanHTTP requires to write in sorted order for the ease of testing.
func (res *Response) WriteSortedHeaders(w io.Writer) error {

	var err error
	bw := bufio.NewWriter(w)
	wtp := textproto.NewWriter(bw)
	keySlice := make([]string, 0)
	for key, _ := range res.Header {
		keySlice = append(keySlice, key)
	}

	// Now sort the slice
	sort.Strings(keySlice)

	// Iterate over all keys in a sorted order
	for _, key := range keySlice {
		fmt.Printf("%s: %s\n", key, res.Header[key])

		if err = wtp.PrintfLine("%s: %s", key, res.Header[key]); err!=nil{
		return err
		}
	}
	if err = wtp.PrintfLine(""); err!=nil{
	return err
	}
	err = bw.Flush()
	return err
}

// WriteBody writes res' file content as the response body to w.
// It doesn't write anything if there is no file to serve.
func (res *Response) WriteBody(w io.Writer) error {
	bw := bufio.NewWriter(w)
	// tpw := textproto.NewWriter(bw)
	if res.FilePath == "" {
		return nil
	} else {
		dat, err := os.ReadFile(res.FilePath)
		if err != nil {
			return err
		}
		if _, err = bw.Write(dat); err != nil {
			return err
		}

	}

	err := bw.Flush()
	return err

}

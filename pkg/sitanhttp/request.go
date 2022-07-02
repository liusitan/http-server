package sitanhttp

import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	// log.Printf("reading request")
	var res Request
	startLine, err := ReadLine(br)
	fmt.Println(startLine)
	if err != nil {
		if len(startLine) > 0 {
			return nil, true, err

		} else {
			return nil, false, err

		}
	} // Read start line
	rerr, method, URI, proto := ReadStartLine(startLine)
	if rerr != nil {
		if (*rerr).statusCode == 400 || (*rerr).statusCode == 404 {
			return nil, true, rerr
		}
	}
	res.Method = method
	res.Proto = proto
	res.URL = URI
	header := make(map[string]string)
	// var header map[string]string
	// Read headers
	var headLine string
	for {
		headLine, err = ReadLine(br)
		fmt.Println(headLine)
		if err != nil {
			return nil, true, rerr
		}
		if len(headLine) == 0 {
			break
		}
		rerr, key, value := parseAPair(headLine)
		if rerr != nil {
			if (*rerr).statusCode == 400 || (*rerr).statusCode == 404 {
				return nil, true, rerr
			}
		}
		header[key] = value

	}
	// headLine, err = ReadLine(br)
	// if err != nil {
	// 	return nil, false, rerr
	// }
	// Check required headers
	if _, ok := header["Host"]; !ok {
		//do something here
			return nil, true, &RequestError{
				info:       "invalid header: Host key is required",
				statusCode: 400,
			}
		
	} else {
		res.Host = header["Host"]
		delete(header, "Host")
	}
	// Handle special headers
	if val, ok := header["Connection"]; ok {
		if val == "close" {
			res.Close = true

		}
		delete(header, "Connection")

	}
	res.Header = header
	return &res, true, nil
}
func parseAPair(line string) (err *RequestError, key string, value string) {
	ci := strings.Index(line, ":")
	if ci <= 0 {
		return &RequestError{
			info:       "invliad header line: no key",
			statusCode: 400,
		}, "", ""
	}
	key = line[:ci]
	if !KeyIsValid(key) {
		return &RequestError{
			info:       "invliad header line: invalid key",
			statusCode: 400,
		}, "", ""
	}
	value = line[ci+1:]
	var vstart int
	for i, c := range value {
		if c != ' ' {
			vstart = i
			break
		}
	}
	value = line[vstart+ci+1:]
	// if len(value) ==0{
	// 	return &RequestError{
	// 		info: "invliad header line: value missing",
	// 		statusCode: 400,
	// 	},"",""
	// }

	return nil, CanonicalHeaderKey(key), value
}
func KeyIsValid(key string) bool {
	for _, r := range key {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func ReadStartLine(line string) (err *RequestError, method string, URI string, proto string) {
	s1 := -1
	s2 := -1
	for i, c := range line {
		if c == ' ' {
			if s1 == -1 {
				s1 = i
			} else if s2 == -1 {
				s2 = i
			} else {
				break
			}
		}
	}

	if s1 < 0 || s2 < 0 {
		return &RequestError{
			info:       "insufficient number of params",
			statusCode: 400,
		}, "", "", ""
	}
	method = line[:s1]
	if method != "GET" {
		return &RequestError{
			info:       "incorrect method " + method,
			statusCode: 400,
		}, "", "", ""
	}
	requestURI := line[s1+1 : s2]
	// err, URI = ParseURI(requestURI)
	URI = requestURI
	if err != nil {
		return err, "", "", ""
	}
	proto = line[s2+1:]
	if proto != "HTTP/1.1" {

		return &RequestError{
			info:       "incorrect method ",
			statusCode: 400,
		}, "", "", ""
	}

	return err, method, URI, proto
}

type RequestError struct {
	info       string
	statusCode int
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("status code: %d %s", e.statusCode, e.info)
}

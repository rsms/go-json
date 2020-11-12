package json

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type Reader struct {
	d          *json.Decoder
	err        error
	tok        json.Token // most recently parsed token
	delimstack []json.Delim
	delim      json.Delim // top of logical delimstack
}

func NewReader(data []byte) *Reader {
	return &Reader{
		d: json.NewDecoder(bytes.NewReader(data)),
	}
}

func (c *Reader) Reset(r io.Reader) {
	c.d = json.NewDecoder(r)
	if c.delimstack != nil {
		c.delimstack = c.delimstack[:0]
	}
}

func (c *Reader) ResetBytes(data []byte) {
	c.Reset(bytes.NewReader(data))
}

func (c *Reader) Err() error { return c.err }

func (c *Reader) setError(err error) {
	if c.err == nil {
		if numerr, ok := err.(*strconv.NumError); ok {
			e := numerr.Unwrap()
			if e == strconv.ErrSyntax {
				err = c.errExpected("number")
			}
		}
		c.err = err
	}
}

func (c *Reader) setErrorf(format string, args ...interface{}) {
	if c.err == nil {
		c.err = fmt.Errorf(format, args...)
	}
}

func (c *Reader) errExpected(expected string) error {
	var actual string
	if d, ok := c.tok.(json.Delim); ok {
		actual = fmt.Sprint(d)
	} else {
		actual = fmt.Sprintf("%T", c.tok)
	}
	return fmt.Errorf("expected %s but got %s at offset %d", expected, actual, c.d.InputOffset())
}

func (c *Reader) setErrorExpected(expected string) {
	c.setError(c.errExpected(expected))
}

func (c *Reader) next() json.Token {
	t, err := c.d.Token()
	c.tok = t
	if err != nil {
		// if err == io.EOF
		c.setError(err)
	}
	return t
}

func (c *Reader) Key() string {
	if c.d.More() {
		t := c.next()
		if s, ok := t.(string); ok {
			return s
		}
		c.setErrorExpected("key")
	}
	return ""
}

func (c *Reader) pushDelim(d json.Delim) bool {
	tok := c.next()
	ok := tok == d
	if !ok {
		c.setErrorExpected(fmt.Sprint(d))
		return false
	}
	c.delimstack = append(c.delimstack, c.delim)
	c.delim = d
	return true
}

func (c *Reader) popDelim() {
	t := c.next()
	if d, ok := t.(json.Delim); ok {
		expect := json.Delim(rune(d) - 2) // i.e. '['+2 = ']', '{'+2 = '}'
		if expect != c.delim {
			// delimiter mismatch, e.g. "[1,2,}"
			c.errExpected(fmt.Sprint(expect))
		}
		c.delim = c.delimstack[len(c.delimstack)-1]
		c.delimstack = c.delimstack[:len(c.delimstack)-1]
	}
}

func (c *Reader) ObjectStart() bool {
	return c.pushDelim(json.Delim('{'))
}

func (c *Reader) ArrayStart() bool {
	return c.pushDelim(json.Delim('['))
}

func (c *Reader) More() bool {
	if c.d.More() {
		return true
	}
	// consume ending delimiter
	c.popDelim()
	return false
}

func (c *Reader) Int(bitsize int) int64 {
	t := c.next()
	switch v := t.(type) {
	case float64:
		return int64(v)
	case string:
		i, err := strconv.ParseInt(v, 10, bitsize)
		if err != nil {
			c.setError(err)
		}
		return i
	default:
		c.setErrorExpected("number")
	}
	return 0
}

func (c *Reader) Uint(bitsize int) uint64 {
	t := c.next()
	switch v := t.(type) {
	case float64:
		return uint64(v)
	case string:
		i, err := strconv.ParseUint(v, 10, bitsize)
		if err != nil {
			c.setError(err)
		}
		return i
	default:
		c.setErrorExpected("number")
	}
	return 0
}

func (c *Reader) Float(bitsize int) float64 {
	t := c.next()
	if v, ok := t.(float64); ok {
		return v
	}
	c.setErrorExpected("number")
	return 0.0
}

func (c *Reader) Bool() bool {
	t := c.next()
	if v, ok := t.(bool); ok {
		return v
	}
	c.setErrorExpected("boolean")
	return false
}

func (c *Reader) Str() string {
	t := c.next()
	if s, ok := t.(string); ok {
		return s
	}
	c.setErrorExpected("string")
	return ""
}

func (c *Reader) Blob() []byte {
	s := c.Str()
	if len(s) == 0 {
		return nil
	}
	buf, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		c.setErrorf("failed to decode blob: %v", err)
	}
	return buf
}

// Discard the next value
func (c *Reader) Discard() {
	switch t := c.next().(type) {
		case json.Delim: // one of [ ] { }
			c.setError(fmt.Errorf("UNIMPLEMENTED json.Reader.Discard object (%q)", t))
	}
}

// Copyright (c) 2016 Uber Technologies, Inc.
// Copyright (c) 2017 Hiroaki Nakamura
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ltsv

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/uber-go/zap"
)

const (
	// For JSON-escaping; see jsonEncoder.safeAddString below.
	_hex = "0123456789abcdef"
	// Initial buffer size for encoders.
	_initialBufSize = 1024
)

var (
	// errNilSink signals that Encoder.WriteEntry was called with a nil WriteSyncer.
	errNilSink = errors.New("can't write encoded message a nil WriteSyncer")
)

var ltsvPool = sync.Pool{New: func() interface{} {
	return &ltsvEncoder{
		bytes:        make([]byte, 0, _initialBufSize),
		timeLabel:    "time",
		levelLabel:   "level",
		messageLabel: "msg",
		replacer: strings.NewReplacer(
			"\t", "\\t",
			"\n", "\\n",
			"\r", "\\r",
		),
	}
}}

type ltsvEncoder struct {
	bytes        []byte
	timeLabel    string
	timeFmt      string
	levelLabel   string
	messageLabel string
	nestedLevel  int
	replacer     *strings.Replacer
}

// NewLTSVEncoder creates a line-oriented LTSV encoder.
// See http://ltsv.org/ for LTSV (Labeled Tab-separated Values).
// By default, the encoder uses RFC3339-formatted timestamps.
// You can change this format with the LTSVTimeFormat option.
//
// The tab \t, newline \n and carriage-return \r are escaped as
// \\t, \\n, \\r respectively.
// You can change this behavior with the LTSVReplacer option.
//
// Nested values are encoded in JSON format.  See Example (Nest).
func NewLTSVEncoder(options ...LTSVOption) zap.Encoder {
	enc := ltsvPool.Get().(*ltsvEncoder)
	enc.truncate()
	enc.timeFmt = time.RFC3339
	for _, opt := range options {
		opt.apply(enc)
	}
	return enc
}

func (enc *ltsvEncoder) Free() {
	ltsvPool.Put(enc)
}

func (enc *ltsvEncoder) AddString(key, val string) {
	enc.addKey(key)
	if enc.nestedLevel > 0 {
		enc.bytes = append(enc.bytes, '"')
		enc.safeAddJSONString(val)
		enc.bytes = append(enc.bytes, '"')
	} else {
		enc.safeAddString(val)
	}
}

func (enc *ltsvEncoder) AddBool(key string, val bool) {
	enc.addKey(key)
	enc.bytes = strconv.AppendBool(enc.bytes, val)
}

func (enc *ltsvEncoder) AddInt(key string, val int) {
	enc.AddInt64(key, int64(val))
}

func (enc *ltsvEncoder) AddInt64(key string, val int64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendInt(enc.bytes, val, 10)
}

func (enc *ltsvEncoder) AddUint(key string, val uint) {
	enc.AddUint64(key, uint64(val))
}

func (enc *ltsvEncoder) AddUint64(key string, val uint64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendUint(enc.bytes, val, 10)
}

func (enc *ltsvEncoder) AddUintptr(key string, val uintptr) {
	enc.addKey(key)
	enc.bytes = append(enc.bytes, "0x"...)
	enc.bytes = strconv.AppendUint(enc.bytes, uint64(val), 16)
}

func (enc *ltsvEncoder) AddFloat64(key string, val float64) {
	enc.addKey(key)
	enc.bytes = strconv.AppendFloat(enc.bytes, val, 'f', -1, 64)
}

func (enc *ltsvEncoder) AddMarshaler(key string, obj zap.LogMarshaler) error {
	enc.addKey(key)
	enc.nestedLevel++
	enc.bytes = append(enc.bytes, '{')
	err := obj.MarshalLog(enc)
	enc.bytes = append(enc.bytes, '}')
	enc.nestedLevel--
	return err
}

func (enc *ltsvEncoder) AddObject(key string, obj interface{}) error {
	enc.AddString(key, fmt.Sprintf("%+v", obj))
	return nil
}

func (enc *ltsvEncoder) Clone() zap.Encoder {
	clone := ltsvPool.Get().(*ltsvEncoder)
	clone.truncate()
	clone.bytes = append(clone.bytes, enc.bytes...)
	clone.timeLabel = enc.timeLabel
	clone.timeFmt = enc.timeFmt
	clone.levelLabel = enc.levelLabel
	clone.messageLabel = enc.messageLabel
	clone.nestedLevel = enc.nestedLevel
	clone.replacer = enc.replacer
	return clone
}

func (enc *ltsvEncoder) WriteEntry(sink io.Writer, msg string, lvl zap.Level, t time.Time) error {
	if sink == nil {
		return errNilSink
	}

	final := ltsvPool.Get().(*ltsvEncoder)
	final.truncate()
	enc.addTime(final, t)
	enc.addLevel(final, lvl)
	enc.addMessage(final, msg)

	if len(enc.bytes) > 0 {
		final.bytes = append(final.bytes, '\t')
		final.bytes = append(final.bytes, enc.bytes...)
	}
	final.bytes = append(final.bytes, '\n')

	expectedBytes := len(final.bytes)
	n, err := sink.Write(final.bytes)
	final.Free()
	if err != nil {
		return err
	}
	if n != expectedBytes {
		return fmt.Errorf("incomplete write: only wrote %v of %v bytes", n, expectedBytes)
	}
	return nil
}

func (enc *ltsvEncoder) truncate() {
	enc.bytes = enc.bytes[:0]
}

func (enc *ltsvEncoder) addKey(key string) {
	if enc.nestedLevel > 0 {
		last := len(enc.bytes) - 1
		// At some point, we'll also want to support arrays.
		if last >= 0 && enc.bytes[last] != '{' {
			enc.bytes = append(enc.bytes, ',')
		}
		enc.bytes = append(enc.bytes, '"')
		enc.safeAddJSONString(key)
		enc.bytes = append(enc.bytes, '"', ':')
	} else {
		if len(enc.bytes) > 0 {
			enc.bytes = append(enc.bytes, '\t')
		}
		enc.safeAddString(key)
		enc.bytes = append(enc.bytes, ':')
	}
}

func (enc *ltsvEncoder) addLevel(final *ltsvEncoder, lvl zap.Level) {
	final.addKey(enc.levelLabel)
	switch lvl {
	case zap.DebugLevel:
		final.bytes = append(final.bytes, 'D')
	case zap.InfoLevel:
		final.bytes = append(final.bytes, 'I')
	case zap.WarnLevel:
		final.bytes = append(final.bytes, 'W')
	case zap.ErrorLevel:
		final.bytes = append(final.bytes, 'E')
	case zap.PanicLevel:
		final.bytes = append(final.bytes, 'P')
	case zap.FatalLevel:
		final.bytes = append(final.bytes, 'F')
	default:
		final.bytes = strconv.AppendInt(final.bytes, int64(lvl), 10)
	}
}

func (enc *ltsvEncoder) addTime(final *ltsvEncoder, t time.Time) {
	if enc.timeFmt == "" {
		return
	}
	final.addKey(enc.timeLabel)
	final.bytes = t.AppendFormat(final.bytes, enc.timeFmt)
}

func (enc *ltsvEncoder) addMessage(final *ltsvEncoder, msg string) {
	final.addKey(enc.messageLabel)
	final.safeAddString(msg)
}

func (enc *ltsvEncoder) safeAddString(s string) {
	enc.bytes = append(enc.bytes, enc.replacer.Replace(s)...)
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's escaping function, it doesn't attempt to
// protect the user from browser vulnerabilities or JSONP-related problems.
func (enc *ltsvEncoder) safeAddJSONString(s string) {
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			i++
			if 0x20 <= b && b != '\\' && b != '"' {
				enc.bytes = append(enc.bytes, b)
				continue
			}
			switch b {
			case '\\', '"':
				enc.bytes = append(enc.bytes, '\\', b)
			case '\n':
				enc.bytes = append(enc.bytes, '\\', 'n')
			case '\r':
				enc.bytes = append(enc.bytes, '\\', 'r')
			case '\t':
				enc.bytes = append(enc.bytes, '\\', 't')
			default:
				// Encode bytes < 0x20, except for the escape sequences above.
				enc.bytes = append(enc.bytes, `\u00`...)
				enc.bytes = append(enc.bytes, _hex[b>>4], _hex[b&0xF])
			}
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			enc.bytes = append(enc.bytes, `\ufffd`...)
			i++
			continue
		}
		enc.bytes = append(enc.bytes, s[i:i+size]...)
		i += size
	}
}

// A LTSVOption is used to set options for a LTSV encoder.
type LTSVOption interface {
	apply(*ltsvEncoder)
}

type ltsvOptionFunc func(*ltsvEncoder)

func (opt ltsvOptionFunc) apply(enc *ltsvEncoder) {
	opt(enc)
}

// LTSVTimeFormat sets the format for log timestamps, using the same layout
// strings supported by time.Parse.
func LTSVTimeFormat(layout string) LTSVOption {
	return ltsvOptionFunc(func(enc *ltsvEncoder) {
		enc.timeFmt = layout
	})
}

// LTSVNoTime omits timestamps from the serialized log entries.
func LTSVNoTime() LTSVOption {
	return LTSVTimeFormat("")
}

// LTSVTimeLabel sets the label for log timestamps.
func LTSVTimeLabel(label string) LTSVOption {
	return ltsvOptionFunc(func(enc *ltsvEncoder) {
		enc.timeLabel = label
	})
}

// LTSVLevelLabel sets the label for log levels.
func LTSVLevelLabel(label string) LTSVOption {
	return ltsvOptionFunc(func(enc *ltsvEncoder) {
		enc.levelLabel = label
	})
}

// LTSVNoLevel omit levels from the serialized log entries.
func LTSVNoLevel() LTSVOption {
	return LTSVLevelLabel("")
}

// LTSVMessageLabel sets the label for log messages.
func LTSVMessageLabel(label string) LTSVOption {
	return ltsvOptionFunc(func(enc *ltsvEncoder) {
		enc.messageLabel = label
	})
}

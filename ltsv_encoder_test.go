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
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/zap"
	"github.com/uber-go/zap/spywrite"
)

var epoch = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

func newLTSVEncoder(opts ...LTSVOption) *ltsvEncoder {
	return NewLTSVEncoder(opts...).(*ltsvEncoder)
}

func withLTSVEncoder(f func(*ltsvEncoder)) {
	enc := newLTSVEncoder()
	f(enc)
	enc.Free()
}

type testBuffer struct {
	bytes.Buffer
}

func (b *testBuffer) Sync() error {
	return nil
}

func (b *testBuffer) Lines() []string {
	output := strings.Split(b.String(), "\n")
	return output[:len(output)-1]
}

func (b *testBuffer) Stripped() string {
	return strings.TrimRight(b.String(), "\n")
}

func assertLTSVOutput(t testing.TB, desc string, expected string, f func(zap.Encoder)) {
	withLTSVEncoder(func(enc *ltsvEncoder) {
		f(enc)
		assert.Equal(t, expected, string(enc.bytes), "Unexpected encoder output after adding a %s.", desc)
	})
	withLTSVEncoder(func(enc *ltsvEncoder) {
		enc.AddString("foo", "bar")
		f(enc)
		expectedPrefix := "foo:bar"
		if expected != "" {
			// If we expect output, it should be tab-separated from the previous
			// field.
			expectedPrefix += "\t"
		}
		assert.Equal(t, expectedPrefix+expected, string(enc.bytes), "Unexpected encoder output after adding a %s as a second field.", desc)
	})
}

func TestLTSVEncoderFields(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
		f        func(zap.Encoder)
	}{
		{"string", "k:v", func(e zap.Encoder) { e.AddString("k", "v") }},
		{"string", "k:", func(e zap.Encoder) { e.AddString("k", "") }},
		{"bool", "k:true", func(e zap.Encoder) { e.AddBool("k", true) }},
		{"bool", "k:false", func(e zap.Encoder) { e.AddBool("k", false) }},
		{"int", "k:42", func(e zap.Encoder) { e.AddInt("k", 42) }},
		{"int64", "k:42", func(e zap.Encoder) { e.AddInt64("k", 42) }},
		{"int64", fmt.Sprintf("k:%d", math.MaxInt64), func(e zap.Encoder) { e.AddInt64("k", math.MaxInt64) }},
		{"uint", "k:42", func(e zap.Encoder) { e.AddUint("k", 42) }},
		{"uint64", "k:42", func(e zap.Encoder) { e.AddUint64("k", 42) }},
		{"uint64", fmt.Sprintf("k:%d", uint64(math.MaxUint64)), func(e zap.Encoder) { e.AddUint64("k", math.MaxUint64) }},
		{"uintptr", "k:0xdeadbeef", func(e zap.Encoder) { e.AddUintptr("k", 0xdeadbeef) }},
		{"float64", "k:1", func(e zap.Encoder) { e.AddFloat64("k", 1.0) }},
		{"float64", "k:10000000000", func(e zap.Encoder) { e.AddFloat64("k", 1e10) }},
		{"float64", "k:NaN", func(e zap.Encoder) { e.AddFloat64("k", math.NaN()) }},
		{"float64", "k:+Inf", func(e zap.Encoder) { e.AddFloat64("k", math.Inf(1)) }},
		{"float64", "k:-Inf", func(e zap.Encoder) { e.AddFloat64("k", math.Inf(-1)) }},
		{"marshaler", `k:{"loggable":"yes"}`, func(e zap.Encoder) {
			assert.NoError(t, e.AddMarshaler("k", loggable{true}), "Unexpected error calling MarshalLog.")
		}},
		{"marshaler", "k:{}", func(e zap.Encoder) {
			assert.Error(t, e.AddMarshaler("k", loggable{false}), "Expected an error calling MarshalLog.")
		}},
		{"ints", "k:[1 2 3]", func(e zap.Encoder) { e.AddObject("k", []int{1, 2, 3}) }},
		{"strings", "k:[bar 1 bar 2 bar 3]",
			func(e zap.Encoder) {
				e.AddObject("k", []string{"bar 1", "bar 2", "bar 3"})
			}},
		{"map[string]string", "k:map[loggable:yes]", func(e zap.Encoder) {
			assert.NoError(t, e.AddObject("k", map[string]string{"loggable": "yes"}), "Unexpected error serializing a map.")
		}},
		{"arbitrary object", "k:{Name:jane}", func(e zap.Encoder) {
			assert.NoError(t, e.AddObject("k", struct{ Name string }{"jane"}), "Unexpected error serializing a struct.")
		}},
	}

	for _, tt := range tests {
		assertLTSVOutput(t, tt.desc, tt.expected, tt.f)
	}
}

func TestLTSVWriteEntry(t *testing.T) {
	entry := &zap.Entry{Level: zap.InfoLevel, Message: "Something happened.", Time: epoch}
	tests := []struct {
		enc      zap.Encoder
		expected string
		name     string
	}{
		{
			enc:      NewLTSVEncoder(),
			expected: "time:1970-01-01T00:00:00Z\tlevel:I\tmsg:Something happened.",
			name:     "RFC822",
		},
		{
			enc:      NewLTSVEncoder(LTSVTimeFormat(time.RFC822)),
			expected: "time:01 Jan 70 00:00 UTC\tlevel:I\tmsg:Something happened.",
			name:     "RFC822",
		},
		{
			enc:      NewLTSVEncoder(LTSVTimeFormat("")),
			expected: "level:I\tmsg:Something happened.",
			name:     "empty layout",
		},
		{
			enc:      NewLTSVEncoder(LTSVNoTime()),
			expected: "level:I\tmsg:Something happened.",
			name:     "NoTime",
		},
	}

	sink := &testBuffer{}
	for _, tt := range tests {
		assert.NoError(
			t,
			tt.enc.WriteEntry(sink, entry.Message, entry.Level, entry.Time),
			"Unexpected failure writing entry with text time formatter %s.", tt.name,
		)
		assert.Equal(t, tt.expected, sink.Stripped(), "Unexpected output from text time formatter %s.", tt.name)
		sink.Reset()
	}
}

func TestLTSVWriteEntryLevels(t *testing.T) {
	tests := []struct {
		level    zap.Level
		expected string
	}{
		{zap.DebugLevel, "D"},
		{zap.InfoLevel, "I"},
		{zap.WarnLevel, "W"},
		{zap.ErrorLevel, "E"},
		{zap.PanicLevel, "P"},
		{zap.FatalLevel, "F"},
		{zap.Level(42), "42"},
	}

	sink := &testBuffer{}
	enc := NewLTSVEncoder(LTSVNoTime())
	for _, tt := range tests {
		assert.NoError(
			t,
			enc.WriteEntry(sink, "Fake message.", tt.level, epoch),
			"Unexpected failure writing entry with level %s.", tt.level,
		)
		expected := fmt.Sprintf("level:%s\tmsg:Fake message.", tt.expected)
		assert.Equal(t, expected, sink.Stripped(), "Unexpected text output for level %s.", tt.level)
		sink.Reset()
	}
}

func TestLTSVClone(t *testing.T) {
	parent := &ltsvEncoder{
		bytes:        make([]byte, 0, 128),
		timeLabel:    "time",
		levelLabel:   "level",
		messageLabel: "msg",
		replacer: strings.NewReplacer(
			"\t", "\\t",
			"\n", "\\n",
			"\r", "\\r",
		),
	}
	clone := parent.Clone()

	// Adding to the parent shouldn't affect the clone, and vice versa.
	parent.AddString("foo", "bar")
	clone.AddString("baz", "bing")

	assert.Equal(t, "foo:bar", string(parent.bytes), "Unexpected serialized fields in parent encoder.")
	assert.Equal(t, "baz:bing", string(clone.(*ltsvEncoder).bytes), "Unexpected serialized fields in cloned encoder.")
}

func TestLTSVWriteEntryFailure(t *testing.T) {
	withLTSVEncoder(func(enc *ltsvEncoder) {
		tests := []struct {
			sink io.Writer
			msg  string
		}{
			{nil, "Expected an error when writing to a nil sink."},
			{spywrite.FailWriter{}, "Expected an error when writing to sink fails."},
			{spywrite.ShortWriter{}, "Expected an error on partial writes to sink."},
		}
		for _, tt := range tests {
			err := enc.WriteEntry(tt.sink, "hello", zap.InfoLevel, time.Unix(0, 0))
			assert.Error(t, err, tt.msg)
		}
	})
}

func TestLTSVTimeOptions(t *testing.T) {
	epoch := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	entry := &zap.Entry{Level: zap.InfoLevel, Message: "Something happened.", Time: epoch}

	enc := NewLTSVEncoder()

	sink := &testBuffer{}
	enc.AddString("foo", "bar")
	err := enc.WriteEntry(sink, entry.Message, entry.Level, entry.Time)
	assert.NoError(t, err, "WriteEntry returned an unexpected error.")
	assert.Equal(
		t,
		"time:1970-01-01T00:00:00Z\tlevel:I\tmsg:Something happened.\tfoo:bar",
		sink.Stripped(),
	)
}

type loggable struct{ bool }

func (l loggable) MarshalLog(kv zap.KeyValue) error {
	if !l.bool {
		return errors.New("can't marshal")
	}
	kv.AddString("loggable", "yes")
	return nil
}

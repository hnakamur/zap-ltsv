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

import "bytes"

// LTSVStack returns stacktrace formatted in one line which can be used as a LTSV value.
func LTSVStack() string {
	return formatStacktraceInOneLine(0, takeStacktrace(nil, false))
}

func formatStacktraceInOneLine(skip int, s string) string {
	buf := []byte(s)

	// This code is copied from
	// https://github.com/hnakamur/ltsvlog/blob/ece22ec10aab08a1795ed376d3799e0fccbd131d/stack.go

	// NOTE: We reuse the same buffer here.
	p := buf[:0]

	for j := 0; j < 1+2*skip; j++ {
		i := bytes.IndexByte(buf, '\n')
		if i == -1 || i+1 > len(buf) {
			goto buffer_too_small
		}
		buf = buf[i+1:]
	}

	for len(buf) > 0 {
		p = append(p, '[')
		i := bytes.IndexByte(buf, '\n')
		if i == -1 {
			goto buffer_too_small
		}
		p = append(p, buf[:i]...)
		p = append(p, ' ')
		if i+2 > len(buf) {
			goto buffer_too_small
		}
		buf = buf[i+2:]
		i = bytes.IndexByte(buf, '\n')
		if i == -1 {
			goto buffer_too_small
		}
		p = append(p, buf[:i]...)
		p = append(p, ']')
		if i+1 > len(buf) {
			goto buffer_too_small
		}
		buf = buf[i+1:]
		if len(buf) > 0 {
			p = append(p, ',')
		}
	}
	return string(p)

buffer_too_small:
	p = append(p, buf...)
	p = append(p, "..."...)
	return string(p)
}

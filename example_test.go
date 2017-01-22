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

package ltsv_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/hnakamur/zap-ltsv"
	"github.com/uber-go/zap"
)

func Example() {
	// Log in LTSV, using a reflection-free LTSV encoder. By default, loggers
	// write all InfoLevel and above logs to standard out.
	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
	)

	logger.Warn("Log without structured data...")
	logger.Warn(
		"Or use strongly-typed wrappers to add structured context.",
		zap.String("library", "zap"),
		zap.Duration("latency", time.Nanosecond),
	)

	// Avoid re-serializing the same data repeatedly by creating a child logger
	// with some attached context. That context is added to all the child's
	// log output, but doesn't affect the parent.
	child := logger.With(
		zap.String("user", "jane@test.com"),
		zap.Int("visits", 42),
	)
	child.Error("Oh no!")

	// Output:
	// level:W	msg:Log without structured data...
	// level:W	msg:Or use strongly-typed wrappers to add structured context.	library:zap	latency:1
	// level:E	msg:Oh no!	user:jane@test.com	visits:42
}

func Example_fileOutput() {
	// Create a temporary file to output logs to.
	f, err := ioutil.TempFile("", "log")
	if err != nil {
		panic("failed to create temporary file")
	}
	defer os.Remove(f.Name())

	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
		// Write the logging output to the specified file instead of stdout.
		// Any type implementing zap.WriteSyncer or zap.WriteFlusher can be used.
		zap.Output(f),
	)

	logger.Info("This is an info log.", zap.Int("foo", 42))

	// Sync the file so logs are written to disk, and print the file contents.
	// zap will call Sync automatically when logging at FatalLevel or PanicLevel.
	f.Sync()
	contents, err := ioutil.ReadFile(f.Name())
	if err != nil {
		panic("failed to read temporary file")
	}

	fmt.Println(string(contents))
	// Output:
	// level:I	msg:This is an info log.	foo:42
}

func ExampleNest() {
	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
	)

	// We'd like the logging context to be {"outer":{"inner":42}}
	nest := zap.Nest("outer", zap.Int("inner", 42))
	logger.Info("Logging a nested field.", nest)

	// Output:
	// level:I	msg:Logging a nested field.	outer:{"inner":42}
}

func ExampleNew() {
	// The default logger outputs to standard out and only writes logs that are
	// Info level or higher.
	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
	)

	// The default logger does not print Debug logs.
	logger.Debug("This won't be printed.")
	logger.Info("This is an info log.")

	// Output:
	// level:I	msg:This is an info log.
}

func ExampleTee() {
	// Multiple loggers can be combine using Tee.
	output := zap.Output(os.Stdout)
	logger := zap.Tee(
		zap.New(zap.NewTextEncoder(zap.TextNoTime()), output),
		zap.New(zap.NewJSONEncoder(zap.NoTime()), output),
		zap.New(ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), output),
	)

	logger.Info("this log gets encoded three times, differently", zap.Int("foo", 42))
	// Output:
	// [I] this log gets encoded three times, differently foo=42
	// {"level":"info","msg":"this log gets encoded three times, differently","foo":42}
	// level:I	msg:this log gets encoded three times, differently	foo:42
}

func ExampleMultiWriteSyncer() {
	// To send output to multiple outputs, use MultiWriteSyncer.
	textLogger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
		zap.Output(zap.MultiWriteSyncer(os.Stdout, os.Stdout)),
	)

	textLogger.Info("One becomes two")
	// Output:
	// level:I	msg:One becomes two
	// level:I	msg:One becomes two
}

func ExampleNew_options() {
	// We can pass multiple options to the New method to configure the logging
	// level, output location, or even the initial context.
	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
		zap.DebugLevel,
		zap.Fields(zap.Int("count", 1)),
	)

	logger.Debug("This is a debug log.")
	logger.Info("This is an info log.")

	// Output:
	// level:D	msg:This is a debug log.	count:1
	// level:I	msg:This is an info log.	count:1
}

func ExampleCheckedMessage() {
	logger := zap.New(
		ltsv.NewLTSVEncoder(ltsv.LTSVNoTime()), // drop timestamps in tests
	)

	// By default, the debug logging level is disabled. However, calls to
	// logger.Debug will still allocate a slice to hold any passed fields.
	// Particularly performance-sensitive applications can avoid paying this
	// penalty by using checked messages.
	if cm := logger.Check(zap.DebugLevel, "This is a debug log."); cm.OK() {
		// Debug-level logging is disabled, so we won't get here.
		cm.Write(zap.Int("foo", 42), zap.Stack())
	}

	if cm := logger.Check(zap.InfoLevel, "This is an info log."); cm.OK() {
		// Since info-level logging is enabled, we expect to write out this message.
		cm.Write()
	}

	// Output:
	// level:I	msg:This is an info log.
}

func ExampleNewLTSVEncoder() {
	// An encoder with the default settings.
	ltsv.NewLTSVEncoder()

	// Dropping timestamps is often useful in tests.
	ltsv.NewLTSVEncoder(ltsv.LTSVNoTime())

	// In production, customize the encoder to work with your log aggregation
	// system.
	ltsv.NewLTSVEncoder(
		ltsv.LTSVTimeFormat(time.RFC3339Nano), // log nanoseconds using a format defined for https://golang.org/pkg/time/#Time.Format
		ltsv.LTSVMessageLabel("message"),      // customize the message label
	)
}

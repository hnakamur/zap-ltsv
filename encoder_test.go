package ltsv_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	ltsv "github.com/hnakamur/zap-ltsv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestEncodeEntry(t *testing.T) {
	type subtestCase struct {
		ent    zapcore.Entry
		fields []zapcore.Field
		want   string
	}

	testCases := []struct {
		subtestName string
		cfg         zapcore.EncoderConfig
		cases       []subtestCase
	}{
		{
			subtestName: "no time, level and msg config",
			cfg: func() zapcore.EncoderConfig {
				cfg := ltsv.NewDevelopmentEncoderConfig()
				cfg.TimeKey = ""
				cfg.LevelKey = ""
				cfg.MessageKey = ""
				return cfg
			}(),
			cases: []subtestCase{
				{
					ent:  zapcore.Entry{},
					want: "\n",
				},
				{
					ent: zapcore.Entry{
						Message: "hello, ltsv logger",
					},
					want: "\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.String("user", "alice"),
					},
					want: "user:alice\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.String("user", "alice"),
						zap.String("group", "adm"),
					},
					want: "user:alice\tgroup:adm\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Skip(),
					},
					want: "\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Stringer("stringer", new(someStringer)),
					},
					want: "stringer:some\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Strings("users", []string{"alice", "bob"}),
					},
					want: "users:[\"alice\",\"bob\"]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Time("created_at", time.Date(2017, 5, 3, 21, 9, 11, 980000000, time.UTC)),
					},
					want: fmt.Sprintf("created_at:%s\n", time.Date(2017, 5, 3, 21, 9, 11, 980000000, time.UTC).Local().Format("2006-01-02T15:04:05.000-0700")),
					// NOTE: I would like to time in the specified localation, that is UTC in this case.
					//want: "created_at:2017-05-03T21:09:11.980Z\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Times("period", []time.Time{
							time.Date(2017, 5, 3, 21, 9, 11, 980000000, time.UTC),
							time.Date(2017, 5, 3, 21, 23, 59, 999999999, time.UTC),
						}),
					},
					want: "period:[\"2017-05-03T21:09:11.980Z\",\"2017-05-03T21:23:59.999Z\"]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Uint("a", 0),
						zap.Uint("b", 9876),
					},
					want: "a:0\tb:9876\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Uint16("a", 0),
						zap.Uint16("b", 9876),
					},
					want: "a:0\tb:9876\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Uint16s("a", []uint16{0, 9876}),
					},
					want: "a:[0,9876]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Any("a", []interface{}{2, "foo"}),
					},
					want: "a:[2,\"foo\"]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Complex128("a", 1+2i),
					},
					want: "a:\"1+2i\"\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Duration("a", time.Duration(time.Minute+2*time.Second)),
					},
					want: "a:1m2s\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Error(errors.New("error 1")),
					},
					want: "error:error 1\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Errors("errors", []error{errors.New("error 1"), errors.New("error 2")}),
					},
					want: "errors:[{\"error\":\"error 1\"},{\"error\":\"error 2\"}]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Int("a", 1),
						zap.Namespace("ns"),
						zap.Int("a", 2),
					},
					want: "a:1\tns:{\"a\":2}\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Int("a", 1),
						zap.Namespace("ns1"),
						zap.Int("a", 2),
						zap.Namespace("ns2"),
						zap.Int("a", 3),
					},
					want: "a:1\tns1:{\"a\":2,\"ns2\":{\"a\":3}}\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Int("a", 1),
						zap.Int("b", -1),
						zap.Namespace("ns1"),
						zap.Int("a", 2),
						zap.Int("b", -2),
						zap.Namespace("ns2"),
						zap.Int("a", 3),
						zap.Int("b", -3),
					},
					want: "a:1\tb:-1\tns1:{\"a\":2,\"b\":-2,\"ns2\":{\"a\":3,\"b\":-3}}\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.ByteString("a", []byte("hello")),
					},
					want: "a:hello\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.ByteStrings("a", [][]byte{[]byte("hello"), []byte("goodbye")}),
					},
					want: "a:[\"hello\",\"goodbye\"]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Binary("a", []byte{'\xca', '\xfe'}),
					},
					want: "a:yv4=\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Bool("a", true),
						zap.Bool("b", false),
					},
					want: "a:true\tb:false\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Bools("a", []bool{true, false}),
					},
					want: "a:[true,false]\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Float64("a", 2.39),
						zap.Float64("max", math.MaxFloat64),
						zap.Float64("smallestNonZero", math.SmallestNonzeroFloat64),
					},
					want: "a:2.39\tmax:179769313486231570000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\tsmallestNonZero:0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000005\n",
				},
				{
					ent: zapcore.Entry{},
					fields: []zapcore.Field{
						zap.Uintptr("a", uintptr(2)),
					},
					want: "a:2\n",
				},
			},
		},
		{
			subtestName: "no time and level config",
			cfg: func() zapcore.EncoderConfig {
				cfg := ltsv.NewDevelopmentEncoderConfig()
				cfg.TimeKey = ""
				cfg.LevelKey = ""
				return cfg
			}(),
			cases: []subtestCase{
				{
					ent:  zapcore.Entry{},
					want: "msg:\n",
				},
				{
					ent: zapcore.Entry{
						Message: "hello, ltsv logger",
					},
					want: "msg:hello, ltsv logger\n",
				},
			},
		},
		{
			subtestName: "no time config",
			cfg: func() zapcore.EncoderConfig {
				cfg := ltsv.NewDevelopmentEncoderConfig()
				cfg.TimeKey = ""
				return cfg
			}(),
			cases: []subtestCase{
				{
					ent: zapcore.Entry{
						Level: zapcore.DebugLevel,
					},
					want: "level:debug\tmsg:\n",
				},
				{
					ent: zapcore.Entry{
						Level:   zapcore.InfoLevel,
						Message: "hello, ltsv logger",
					},
					want: "level:info\tmsg:hello, ltsv logger\n",
				},
			},
		},
		{
			subtestName: "developer config",
			cfg:         ltsv.NewDevelopmentEncoderConfig(),
			cases: []subtestCase{
				{
					ent: zapcore.Entry{
						Level: zapcore.DebugLevel,
						Time:  time.Date(2017, 5, 3, 21, 9, 11, 983000000, time.UTC),
					},
					want: "time:2017-05-03T21:09:11.983Z\tlevel:debug\tmsg:\n",
				},
				{
					ent: zapcore.Entry{
						Level:   zapcore.InfoLevel,
						Time:    time.Date(2017, 5, 3, 21, 9, 11, 980000000, time.UTC),
						Message: "hello, ltsv logger",
					},
					want: "time:2017-05-03T21:09:11.980Z\tlevel:info\tmsg:hello, ltsv logger\n",
				},
			},
		},
	}
	for _, tc := range testCases {
		enc := ltsv.NewLTSVEncoder(tc.cfg)
		t.Run(tc.subtestName, func(t *testing.T) {
			for _, st := range tc.cases {
				buf, err := enc.EncodeEntry(st.ent, st.fields)
				if err != nil {
					t.Fatalf("failed to encode entry; ent=%+v, fields=%+v, err=%+v", st.ent, st.fields, err)
				}
				got := buf.String()
				if got != st.want {
					t.Errorf("got=%q, want=%q, ent=%+v, fields=%+v", got, st.want, st.ent, st.fields)
				}
			}
		})
	}
}

type someStringer struct{}

func (s *someStringer) String() string {
	return "some"
}

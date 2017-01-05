package main

import (
	"time"

	"github.com/hnakamur/zap-ltsv"
	"github.com/uber-go/zap"
)

type Auth struct {
	ExpiresAt time.Time `json:"expires_at"`
	// Since we'll need to send the token to the browser, we include it in the
	// struct's JSON representation.
	Token string `json:"token"`
}

func (a Auth) MarshalLog(kv zap.KeyValue) error {
	kv.AddInt64("expires_at", a.ExpiresAt.UnixNano())
	// We don't want to log sensitive data.
	kv.AddString("token", "---")
	return nil
}

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	Auth Auth   `auth:"auth"`
}

func (u User) MarshalLog(kv zap.KeyValue) error {
	kv.AddString("name", u.Name)
	kv.AddInt("age", u.Age)
	return kv.AddMarshaler("auth", u.Auth)
}

func main() {
	logger := zap.New(
		ltsv.NewLTSVEncoder(),
	)
	logger.Warn("Log without structured data...")
	logger.Warn("Log without structured data\nwith newline...")
	logger.Warn(
		"Or use strongly-typed wrappers to add structured context.",
		zap.String("library", "zap"),
		zap.Duration("latency", time.Nanosecond),
	)

	child := logger.With(
		zap.String("user", "jane@test.com"),
		zap.Int("visits", 42),
	)
	child.Error("Oh no!")
	child.Info("Yes!")

	jane := User{
		Name: "Jane Doe",
		Age:  42,
		Auth: Auth{
			ExpiresAt: time.Unix(0, 100),
			Token:     "super secret",
		},
	}

	logger.Info("Successful login.", zap.Marshaler("user", jane))
}

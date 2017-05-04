package ltsv_test

import (
	"time"

	ltsv "github.com/hnakamur/zap-ltsv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Example() {
	err := ltsv.RegisterLTSVEncoder()
	if err != nil {
		panic(err)
	}

	logger, err := ltsv.NewDevelopmentConfig().Build()
	if err != nil {
		panic(err)
	}

	logger.Error(
		"use strongly-typed wrappers to add structured context.",
		zap.String("library", "zap"),
		zap.Duration("latency", time.Nanosecond),
	)

	//Output:

	//NOTE: Actually we don't test the result because the time part varys.
}

type user struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (u user) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", u.Name)
	enc.AddString("email", u.Email)
	enc.AddInt64("created_at", u.CreatedAt.UnixNano())
	return nil
}

var jane = user{
	Name:      "Jane Doe",
	Email:     "jane@test.com",
	CreatedAt: time.Date(1980, 1, 1, 12, 0, 0, 0, time.UTC),
}

type users []user

func (u users) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, user := range u {
		err := enc.AppendObject(user)
		if err != nil {
			return err
		}
	}
	return nil
}

func Example_reflect() {
	// type user struct {
	// 	Name      string    `json:"name"`
	// 	Email     string    `json:"email"`
	// 	CreatedAt time.Time `json:"created_at"`
	// }
	//
	// var jane = user{
	// 	Name:      "Jane Doe",
	// 	Email:     "jane@test.com",
	// 	CreatedAt: time.Date(1980, 1, 1, 12, 0, 0, 0, time.UTC),
	// }
	//
	// type users []user
	//
	// err := ltsv.RegisterLTSVEncoder()
	// if err != nil {
	// 	panic(err)
	// }

	logger, err := ltsv.NewDevelopmentConfig().Build()
	if err != nil {
		panic(err)
	}

	logger.Info(
		"test array",
		zap.String("pacakge", "github.com/hnakamur/zap-ltsv"),
		zap.String("backslash", `a	b`),
		zap.Array("users", users{jane}),
	)

	//Output:

	//NOTE: Actually we don't test the result because the time part varys.
}

func Example_nested() {
	// type user struct {
	// 	Name      string    `json:"name"`
	// 	Email     string    `json:"email"`
	// 	CreatedAt time.Time `json:"created_at"`
	// }
	//
	// func (u user) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// 	enc.AddString("name", u.Name)
	// 	enc.AddString("email", u.Email)
	// 	enc.AddInt64("created_at", u.CreatedAt.UnixNano())
	// 	return nil
	// }
	//
	// var jane = user{
	// 	Name:      "Jane Doe",
	// 	Email:     "jane@test.com",
	// 	CreatedAt: time.Date(1980, 1, 1, 12, 0, 0, 0, time.UTC),
	// }
	//
	// type users []user
	//
	// func (u users) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	// 	for _, user := range u {
	// 		err := enc.AppendObject(user)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// }
	//
	// err := ltsv.RegisterLTSVEncoder()
	// if err != nil {
	// 	panic(err)
	// }

	logger, err := ltsv.NewDevelopmentConfig().Build()
	if err != nil {
		panic(err)
	}

	logger.Info(
		"test array",
		zap.String("pacakge", "github.com/hnakamur/zap-ltsv"),
		zap.String("backslash", `a	b`),
		zap.Array("users", users{jane}),
	)

	//Output:

	//NOTE: Actually we don't test the result because the time part varys.
}

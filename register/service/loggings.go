package service

import (
	"github.com/go-kit/kit/log"
	"time"
)

// loggingMiddleware Make a new type
// that contains Service interface and logger instance
type loggingMiddleware struct {
	UserService
	logger log.Logger
}

// LoggingMiddleware returns a logging middleware
func LoggingMiddleware(logger log.Logger) UserServiceMiddleware {
	return func(next UserService) UserService {
		return loggingMiddleware{next, logger}
	}
}

func (mw loggingMiddleware) Login(username, password string) (ok bool, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "login",
			"username", username,
			"password", password,
			"result", ok,
			"took", time.Since(begin),
		)
	}(time.Now())

	ok, err = mw.UserService.Login(username, password)
	return
}

func (mw loggingMiddleware) SignUp(username, password string) (ok bool, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "login",
			"username", username,
			"password", password,
			"result", ok,
			"took", time.Since(begin),
		)
	}(time.Now())

	ok, err = mw.UserService.Login(username, password)
	return
}

func (mw loggingMiddleware) HealthCheck() (result bool) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"function", "HealthCheck",
			"result", result,
			"took", time.Since(begin),
			)
	}(time.Now())

	result = mw.UserService.HealthCheck()
	return
}

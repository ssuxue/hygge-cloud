package service

import (
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/juju/ratelimit"
	"time"
)

var ErrLimitExceed = errors.New("rate limit exceed")

// create limiter middleware with juju/ratelimit
func NewTokenBucketLimiter(bkt *ratelimit.Bucket) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if bkt.TakeAvailable(1) == 0 {
				return nil, ErrLimitExceed
			}
			return next(ctx, request)
		}
	}
}

// Define monitoring middleware, embed UserService
// Add two monitoring indicators: requestCount and requestLatency.
type metricMiddle struct {
	UserService
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

// Get indicators' method.
func Metrics(requestCount metrics.Counter, requestLatency metrics.Histogram) UserServiceMiddleware {
	return func(next UserService) UserService {
		return metricMiddle{
			next,
			requestCount,
			requestLatency,
		}
	}
}

func (mw metricMiddle) Login(username, password string) (ok bool, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "login"}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	ok, err = mw.Login(username, password)
	return
}

func (mw metricMiddle) SignUp(username, password string) (token string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "signup"}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	token, err = mw.SignUp(username, password)
	return
}

func (mw metricMiddle) HealthCheck() (result bool) {
	defer func(begin time.Time) {
		lvs := []string{"method", "health check"}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	result = mw.UserService.HealthCheck()
	return
}

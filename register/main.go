package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/juju/ratelimit"
	"hygge-cloud/register/endpoints"
	"hygge-cloud/register/service"
	"hygge-cloud/register/transports"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 定义环境变量
	var (
		consulHost  = flag.String("consul.host", "localhost", "consul ip address")
		consulPort  = flag.String("consul.port", "8500", "consul port")
		serviceHost = flag.String("service.host", "192.168.124.9", "service ip address")
		servicePort = flag.String("service.port", "9000", "service port")
	)

	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var svc service.UserService
	svc = service.NewMemberService()
	svc = service.LoggingMiddleware(logger)(svc)
	endpoint := endpoints.MakeEndpoint(svc)

	// Set limiter capacity 3
	rateBucket := ratelimit.NewBucket(time.Second, 3)
	endpoint = service.NewTokenBucketLimiter(rateBucket)(endpoint)

	healthEndpoint := endpoints.MakeHealthCheckEndpoint(svc)
	endpts := endpoints.MemberEndpoints{
		MemberEndpoint:      endpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	router := transports.MakeHttpHandler(ctx, endpts, logger)

	// create register object
	register := transports.Register(*consulHost, *consulPort, *serviceHost, *servicePort, logger)

	go func() {
		fmt.Println("Http Server start at port:" + *servicePort)
		// Execute register before running.
		register.Register()
		errChan <- http.ListenAndServe(":" + *servicePort, router)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	err := <-errChan
	// Exit server and cancel register
	register.Deregister()
	fmt.Println(err)
}

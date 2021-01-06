package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	zp "github.com/go-kit/kit/tracing/zipkin"
	"github.com/juju/ratelimit"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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
		zipkinURL   = flag.String("zipkin.url", "http://192.168.124.9:9411/api/v2/spans", "Zipkin server url")
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

	fieldKeys := []string{"method"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "chase",
		Subsystem: "hygge_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)

	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "chase",
		Subsystem: "hygge_service",
		Name:      "request_latency",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	var svc service.UserService
	svc = service.NewMemberService()
	svc = service.LoggingMiddleware(logger)(svc)
	svc = service.Metrics(requestCount, requestLatency)(svc)
	endpoint := endpoints.MakeEndpoint(svc)

	var zipkinTracer *zipkin.Tracer
	{
		var (
			err           error
			hostPort      = *serviceHost + ":" + *servicePort
			serviceName   = "hygge-service"
			useNoopTracer = (*zipkinURL == "")
			reporter      = zipkinhttp.NewReporter(*zipkinURL)
		)
		defer reporter.Close()
		zEP, _ := zipkin.NewEndpoint(serviceName, hostPort)
		zipkinTracer, err = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(zEP), zipkin.WithNoopTracer(useNoopTracer),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		if !useNoopTracer {
			logger.Log("tracer", "Zipkin", "type", "Native", "URL", *zipkinURL)
		}
	}

	// Set limiter capacity 3
	rateBucket := ratelimit.NewBucket(time.Second, 3)
	endpoint = service.NewTokenBucketLimiter(rateBucket)(endpoint)
	endpoint = zp.TraceEndpoint(zipkinTracer, "login-endpoint")(endpoint)

	healthEndpoint := endpoints.MakeHealthCheckEndpoint(svc)
	healthEndpoint = service.NewTokenBucketLimiter(rateBucket)(healthEndpoint)
	healthEndpoint = zp.TraceEndpoint(zipkinTracer, "health-endpoint")(healthEndpoint)

	endpts := endpoints.MemberEndpoints{
		MemberEndpoint:      endpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	router := transports.MakeHttpHandler(ctx, endpts, zipkinTracer, logger)

	// create register object
	register := transports.Register(*consulHost, *consulPort, *serviceHost, *servicePort, logger)

	go func() {
		fmt.Println("Http Server start at port:" + *servicePort)
		// Execute register before running.
		register.Register()
		errChan <- http.ListenAndServe(":"+*servicePort, router)
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

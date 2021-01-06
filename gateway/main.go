package main

import (
	"flag"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"github.com/openzipkin/zipkin-go"
	zipkinhttpsvr "github.com/openzipkin/zipkin-go/middleware/http"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"hygge-cloud/gateway/router"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Create a reverse proxy processing method.
func NewReverseProxy(client *api.Client, zikkinTracer *zipkin.Tracer, logger log.Logger) *httputil.ReverseProxy {

	// create director
	director := func(req *http.Request) {
		// Query the origin`````````````````````````````````````````````````````al request path, such as: /hygge/signInOrUp/login/chase/123.
		// The hygge is service name and the another is interface path.
		reqPath := req.URL.Path
		if reqPath == "" {
			return
		}

		// 按照分隔符'/'对路径进行分解，获取服务名称serviceName
		pathArray := strings.Split(reqPath, "/")
		svcName := pathArray[1]

		// 调用consul api查询serviceName的服务实例列表
		result, _, err := client.Catalog().Service(svcName, "", nil)
		if err != nil {
			logger.Log("ReverseProxy failed.", "Query service instance error: ", err.Error())
			return
		}

		if len(result) == 0 {
			logger.Log("ReverseProxy failed.", "No such service instance: ", svcName)
			return
		}

		//重新组织请求路径，去掉服务名称部分
		destPath := strings.Join(pathArray[2:], "/")

		// 随机选择一个服务实例
		tgt := result[rand.Int()%len(result)]
		logger.Log("service id: ", tgt.ServiceID)

		// 设置代理服务地址信息
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s:%d", tgt.ServiceAddress, tgt.ServicePort)
		req.URL.Path = "/" + destPath
	}

	// 为反向代理增加追踪逻辑，使用如下RoundTrip代替默认Transport
	roundTrip, _ := zipkinhttpsvr.NewTransport(zikkinTracer, zipkinhttpsvr.TransportTrace(true))

	return &httputil.ReverseProxy{
		Director:  director,
		Transport: roundTrip,
	}
}

func main() {
	// 创建环境变量
	var (
		consulHost = flag.String("consul.host", "192.168.124.9", "consul server ip address")
		consulPort = flag.String("consul.port", "8500", "consul server port")
		zipkinURL  = flag.String("zipkin.url", "http://192.168.124.9:9411/api/v2/spans", "Zipkin server url")
	)
	flag.Parse()

	// 创建日志组件
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var zipkinTracer *zipkin.Tracer
	{
		var (
			err           error
			hostPort      = "localhost:9090"
			svcName       = "gateway-service"
			useNoopTracer = (*zipkinURL == "")
			reporter      = zipkinhttp.NewReporter(*zipkinURL)
		)

		defer reporter.Close()

		zipkinEndp, _ := zipkin.NewEndpoint(svcName, hostPort)
		zipkinTracer, err = zipkin.NewTracer(
			reporter,
			zipkin.WithLocalEndpoint(zipkinEndp),
			zipkin.WithNoopTracer(useNoopTracer),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}

		if !useNoopTracer {
			logger.Log("tracer", "Zipkin", "type", "Native", "URL", *zipkinURL)
		}
	}

	// Create consul api client.
	consulCfg := api.DefaultConfig()
	consulCfg.Address = "http://" + *consulHost + ":" + *consulPort
	consulCli, err := api.NewClient(consulCfg)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	// Create Reverse Proxy
	// proxy := NewReverseProxy(consulCli, zipkinTracer, logger)

	tags := map[string]string{
		"component": "gateway_server",
	}

	hystrixRouter := router.Routes(consulCli, zipkinTracer, "Circuit Breaker:Service unavailable", logger)

	handler := zipkinhttpsvr.NewServerMiddleware(
		zipkinTracer,
		zipkinhttpsvr.SpanName("gateway"),
		zipkinhttpsvr.TagResponseSize(true),
		zipkinhttpsvr.ServerTags(tags),
	)(hystrixRouter)

	errChan := make(chan error)

	// 启用hystrix实时监控，监听端口为9010
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go func() {
		errChan <- http.ListenAndServe(net.JoinHostPort("", "9010"), hystrixStreamHandler)
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	//开始监听
	go func() {
		logger.Log("transport", "HTTP", "addr", "9090")
		errChan <- http.ListenAndServe(":9090", handler)
	}()

	// 开始运行，等待结束
	logger.Log("exit", <-errChan)
}

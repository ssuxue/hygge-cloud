package router

import (
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"github.com/openzipkin/zipkin-go"
	zipkinhttpsvr "github.com/openzipkin/zipkin-go/middleware/http"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
)

type HystrixRouter struct {
	svcMap       *sync.Map      // 服务实例，存储已经通过hystrix监控服务列表
	logger       log.Logger     // 日志工具
	fallbackMsg  string         // 回调消息
	consulClient *api.Client    // consul客户端对象
	tracer       *zipkin.Tracer // 服务追踪对象
}

func Routes(client *api.Client, tracer *zipkin.Tracer, fbMsg string, logger log.Logger) http.Handler {
	return &HystrixRouter{
		svcMap:       &sync.Map{},
		logger:       logger,
		fallbackMsg:  fbMsg,
		consulClient: client,
		tracer:       tracer,
	}
}

func (router *HystrixRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Query the original request path, such as: /hygge/signInOrUp/login/chase/123.
	// The hygge is service name and the another is interface path.
	reqPath := r.URL.Path
	if reqPath == "" {
		return
	}

	// 按照分隔符'/'对路径进行分解，获取服务名称serviceName
	pathArray := strings.Split(reqPath, "/")
	svcName := pathArray[1]

	// check if service is in monitor
	if _, ok := router.svcMap.Load(svcName); !ok {
		// Take svcName as the command object and set the parameters.
		hystrix.ConfigureCommand(svcName, hystrix.CommandConfig{Timeout: 1000})
		router.svcMap.Store(svcName, svcName)
	}

	// execute the command
	err := hystrix.Do(svcName, func() (err error) {
		// 调用consul api查询serviceName的服务实例列表
		result, _, err := router.consulClient.Catalog().Service(svcName, "", nil)
		if err != nil {
			router.logger.Log("ReverseProxy failed.", "Query service instance error: ", err.Error())
			return
		}

		if len(result) == 0 {
			router.logger.Log("ReverseProxy failed.", "No such service instance: ", svcName)
			return errors.New("no such service instance")
		}

		director := func(req *http.Request) {
			//重新组织请求路径，去掉服务名称部分
			destPath := strings.Join(pathArray[2:], "/")

			// 随机选择一个服务实例
			tgt := result[rand.Int()%len(result)]
			router.logger.Log("service id: ", tgt.ServiceID)

			// 设置代理服务地址信息
			req.URL.Scheme = "http"
			req.URL.Host = fmt.Sprintf("%s:%d", tgt.ServiceAddress, tgt.ServicePort)
			req.URL.Path = "/" + destPath
		}

		var proxyErr error = nil
		// 为反向代理增加追踪逻辑，使用如下RoundTrip代替默认Transport
		roundTrip, _ := zipkinhttpsvr.NewTransport(router.tracer, zipkinhttpsvr.TransportTrace(true))

		// 反向代理失败时错误处理
		errHandler := func(ew http.ResponseWriter, er *http.Request, err error) {
			proxyErr = err
		}

		proxy := &httputil.ReverseProxy{
			Director:     director,
			Transport:    roundTrip,
			ErrorHandler: errHandler,
		}
		proxy.ServeHTTP(w, r)

		return proxyErr

	}, func(err error) error {
		// Running errors, return fallback message.
		router.logger.Log("Fallback errors description: ", err.Error())
		return errors.New(router.fallbackMsg)
	})

	// Do方法执行失败，响应错误信息
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
}

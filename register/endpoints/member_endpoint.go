package endpoints

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"hygge-cloud/register/service"
	"strings"
)

type MemberRequest struct {
	//Id          int    `json:"id"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	RequestType string `json:"request_type"`
}

type MemberResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type HealthRequest struct {
}

type HealthResponse struct {
	Status bool `json:"status"`
}

type MemberEndpoints struct {
	MemberEndpoint      endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

func MakeEndpoint(svc service.UserService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(MemberRequest)
		username := req.Username
		password := req.Password

		if strings.EqualFold(req.RequestType, "login") {
			ok, err := svc.Login(username, password)
			if err != nil {
				return nil, fmt.Errorf("login error: %v", err)
			}
			if !ok {
				return nil, errors.New("username or password is wrong")
			}
		} else if strings.EqualFold(req.RequestType, "signup") {
			// TODO 注册
			token, err := svc.SignUp(username, password)
			if err != nil {
				return "", fmt.Errorf("signup error: %v", err)
			}
			return MemberResponse{Code: 200, Message: "操作成功", Data: token}, nil
		} else {
			return nil, errors.New("incorrect request mode")
		}

		return MemberResponse{Code: 200, Message: "操作成功", Data: req}, nil
	}
}

// Creating health check endpoint.
func MakeHealthCheckEndpoint(svc service.UserService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthResponse{status}, nil
	}
}

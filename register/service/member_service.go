package service

import "errors"

type UserService interface {
	Login(username, password string) (bool, error)
	SignUp(username, password string) (bool, error)
	HealthCheck() bool
}

type MemberService struct {
}

func NewMemberService() *MemberService {
	return &MemberService{}
}

func (MemberService) Login(username, password string) (bool, error) {
	if username == "chase" && password == "123" {
		return true, nil
	}

	if username == "" || password == "" {
		return false, errors.New("parameters can not be empty")
	}

	return false, errors.New("username or password is wrong")
}

func (MemberService) SignUp(username, password string) (bool, error) {
	if username == "" || password == "" {
		return false, errors.New("parameters can not be empty")
	}

	return true, nil
}

// 用于检查服务的健康状态，这里仅仅返回true。
func (MemberService) HealthCheck() bool {
	return true
}

// This defines member service middleware.
type UserServiceMiddleware func(UserService) UserService

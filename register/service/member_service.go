package service

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type UserService interface {
	Login(username, password string) (bool, error)
	SignUp(username, password string) (string, error)
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

func (MemberService) SignUp(username, password string) (string, error) {
	if username == "" || password == "" {
		return "", errors.New("parameters can not be empty")
	}

	// 为了演示方便，设置两分钟后过期
	expAt := time.Now().Add(time.Duration(2) * time.Minute).Unix()

	// 创建声明
	claims := UserCustomClaims{
		Username: username,
		Password: password,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expAt,
			Issuer:    "system",
		},
	}

	//创建token，指定加密算法为HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//生成token
	return token.SignedString(secretKey)
}

// 用于检查服务的健康状态，这里仅仅返回true。
func (MemberService) HealthCheck() bool {
	return true
}

// This defines member service middleware.
type UserServiceMiddleware func(UserService) UserService

//secret key
var secretKey = []byte("suxue1234!@#$")

// ArithmeticCustomClaims 自定义声明
type UserCustomClaims struct {
	Username string `json:"username"`
	Password string `json:"password"`

	jwt.StandardClaims
}

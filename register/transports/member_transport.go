package transports

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"hygge-cloud/register/endpoints"
	"net/http"
)

var ErrorBadRequest = errors.New("invalid request parameter")

func decodeMemberRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	requestType, ok := vars["type"]
	if !ok {
		return nil, ErrorBadRequest
	}

	username, ok := vars["username"]
	if !ok {
		return nil, ErrorBadRequest
	}

	password, ok := vars["password"]
	if !ok {
		return nil, ErrorBadRequest
	}

	return endpoints.MemberRequest{
		Username:    username,
		Password:    password,
		RequestType: requestType,
	}, nil
}

// decodeHealthCheckRequest decode request
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoints.HealthRequest{}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoint endpoints.MemberEndpoints, logger log.Logger) http.Handler {
	r := mux.NewRouter()

	options := []httptransport.ServerOption{
		// Deprecated: Use ServerErrorHandler instead.
		httptransport.ServerErrorLogger(logger),
		httptransport.ServerErrorEncoder(httptransport.DefaultErrorEncoder),
	}

	r.Methods("POST").Path("/signInOrUp/{type}/{username}/{password}").Handler(httptransport.NewServer(
		endpoint.MemberEndpoint,
		decodeMemberRequest,
		encodeResponse,
		options...,
	))

	r.Path("/metrics").Handler(promhttp.Handler())

	r.Methods("GET").Path("/health").Handler(httptransport.NewServer(
		endpoint.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeResponse,
		options...,
	))

	return r
}

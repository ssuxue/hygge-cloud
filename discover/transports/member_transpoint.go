package transports

import (
	"context"
	"encoding/json"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"hygge-cloud/register/endpoints"
	"net/http"
)

// 这里必须要用register中通过一个结构体，不然会报错:types from different packages
//type MemberRequest struct {
//	Username    string `json:"username"`
//	Password    string `json:"password"`
//	RequestType string `json:"request_type"`
//}
//
//type MemberResponse struct {
//	Code    int         `json:"code"`
//	Message string      `json:"message"`
//	Data    interface{} `json:"data"`
//}

// decodeHealthCheckRequest decode request
func decodeDiscoverRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request endpoints.MemberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeDiscoverResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// MakeHttpHandler makes http handler use mux
func MakeHttpHandler(endpoint endpoint.Endpoint) http.Handler {
	r := mux.NewRouter()

	r.Methods("POST").Path("/signInOrUp").Handler(httptransport.NewServer(
		endpoint,
		decodeDiscoverRequest,
		encodeDiscoverResponse,
	))

	return r
}

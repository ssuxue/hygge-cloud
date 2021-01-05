package transports

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	httptransport "github.com/go-kit/kit/transport/http"
	"hygge-cloud/register/endpoints"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func MemberFactory(ctx context.Context, method, path string) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "http://" + instance
		}

		target, err := url.Parse(instance)
		if err != nil {
			return nil, nil, err
		}

		target.Path = path
		var (
			enc httptransport.EncodeRequestFunc
			dec httptransport.DecodeResponseFunc
		)
		enc, dec = encodeMemberRequest, decodeResponse

		return httptransport.NewClient(method, target, enc, dec).Endpoint(), nil, nil
	}
}
func encodeMemberRequest(_ context.Context, req *http.Request, request interface{}) error {
	memberReq := request.(endpoints.MemberRequest)
	path := "/" + memberReq.RequestType + "/" + memberReq.Username + "/" + memberReq.Password
	req.URL.Path += path
	return nil
}

func decodeResponse(_ context.Context, resp *http.Response) (interface{}, error) {
	var response endpoints.MemberResponse
	var s map[string]interface{}

	if respCode := resp.StatusCode; respCode >= 400 {
		if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
			return nil, err
		}
		return nil, errors.New(s["error"].(string) + "\n")
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

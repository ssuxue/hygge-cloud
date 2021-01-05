package endpoints

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"hygge-cloud/discover/transports"
	"time"
)

// using consul.Client to create endpoint discovery
func MakeDiscover(ctx context.Context, client consul.Client, logger log.Logger) endpoint.Endpoint {
	svcName := "hygge"
	tags := []string{"hygge", "chase"}
	passingOnly := true
	duration := 500 * time.Millisecond

	instancer := consul.NewInstancer(client, logger, svcName, tags, passingOnly)

	// Creating interface sd.Factory to function "Login".
	factory := transports.MemberFactory(ctx, "POST", "signInOrUp")

	// Using consul connects server.(Discover server.)
	// Factory creates sd.Factory.
	endpointer := sd.NewEndpointer(instancer, factory, logger)

	// Creating RoundRibbon load balancer
	balancer := lb.NewRoundRobin(endpointer)

	// Add retry functionality to load balancer.
	retry := lb.Retry(1, duration, balancer)

	return retry
}

package downstream

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/ygo-skc/skc-go/common/ygo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	ygoServiceClient ygo.CardServiceClient
)

func init() {
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: false,
		ServerName:         "ygo-service.skc.cards",
	})

	conn, err := grpc.NewClient("ygo-service:9020",
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.UseCompressor("gzip"),
		),
		grpc.WithDefaultServiceConfig(`{
			"methodConfig": [{
				"name": [{"service": "ygo.CardService"}],
				"retryPolicy": {
					"MaxAttempts": 3,
					"InitialBackoff": "0.1s",
					"MaxBackoff": "1s",
					"BackoffMultiplier": 2.0,
					"RetryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}]
		}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             1 * time.Second,
			PermitWithoutStream: true,
		}))

	if err != nil {
		log.Fatalf("Could not connect to ygo-service: %v", err)
	}

	ygoServiceClient = ygo.NewCardServiceClient(conn)
}

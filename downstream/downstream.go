package downstream

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/ygo-skc/skc-go/common/ygo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
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

	_ = gzip.Name

	conn, err := grpc.NewClient("ygo-service:9020",
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	ygoServiceClient = ygo.NewCardServiceClient(conn)
}

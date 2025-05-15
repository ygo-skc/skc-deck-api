package downstream

import (
	"log"

	"github.com/ygo-skc/skc-go/common/ygo"
)

var (
	cardServiceClient ygo.CardServiceClient
)

func init() {
	if client, err := ygo.CreateCardServiceClient("ygo-service.skc.cards", "ygo-service:9020"); err != nil {
		log.Fatalf("Could not connect to ygo-service: %v", err)
	} else {
		cardServiceClient = *client
	}
}

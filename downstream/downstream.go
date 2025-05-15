package downstream

import (
	"log"

	"github.com/ygo-skc/skc-go/common/client"
	"github.com/ygo-skc/skc-go/common/ygo"
)

var (
	CardServiceClient ygo.CardServiceClient
)

func init() {
	if client, err := client.CreateCardServiceClient("ygo-service.skc.cards", "ygo-service:9020"); err != nil {
		log.Fatalf("Failed to connect to ygo-service: %v", err)
	} else {
		CardServiceClient = *client
	}
}

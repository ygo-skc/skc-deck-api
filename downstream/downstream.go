package downstream

import (
	"log"

	"github.com/ygo-skc/skc-go/common/client"
	cUtil "github.com/ygo-skc/skc-go/common/util"
)

var (
	YGO client.YGOClientImpV1
)

func ConnectToYGOService() {
	if c, err := client.NewYGOServiceClients("ygo-service.skc.cards", cUtil.EnvMap["YGO_SERVICE_HOST"]); err != nil {
		log.Fatalf("Failed to connect to ygo-service: %v", err)
	} else {
		YGO = *c
	}
}

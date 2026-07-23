package downstream

import (
	"log/slog"
	"os"

	"github.com/ygo-skc/skc-go/common/v3/client"
	cUtil "github.com/ygo-skc/skc-go/common/v3/util"
)

var (
	YGO client.YGOClientImpV1
)

func ConnectToYGOService() {
	if c, err := client.NewYGOServiceClients("ygo-service.skc.cards", cUtil.EnvMap["YGO_SERVICE_HOST"]); err != nil {
		slog.Error("Failed to connect to ygo-service", "err", err)
		os.Exit(1)
	} else {
		YGO = *c
	}
}

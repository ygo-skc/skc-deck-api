package main

import (
	"os"
	"strings"

	"github.com/ygo-skc/skc-deck-api/api"
	"github.com/ygo-skc/skc-deck-api/db"
	cUtil "github.com/ygo-skc/skc-go/common/util"
	_ "google.golang.org/grpc/encoding/gzip"
)

const (
	ENV_VARIABLE_NAME string = "SKC_DECK_API_DOT_ENV_FILE"
)

func init() {
	isCICD := os.Getenv("IS_CICD")
	if isCICD != "true" && !strings.HasSuffix(os.Args[0], ".test") {
		cUtil.ConfigureEnv(ENV_VARIABLE_NAME)
	}
}

func main() {
	db.EstablishSKCDeckAPIDBConn()
	go api.RunHttpServer()
	select {}
}

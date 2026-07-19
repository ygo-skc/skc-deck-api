package api

import (
	"encoding/json"
	"net/http"

	cModel "github.com/ygo-skc/skc-go/common/v2/model"
	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
)

const (
	statusOp = "Status"
)

// Handler for status/health check endpoint of the api.
// Will get status of downstream services as well to help isolate problems.
func getAPIStatusHandler(res http.ResponseWriter, req *http.Request) {
	logger, ctx := cUtil.InitRequest(req.Context(), apiName, statusOp)

	downstreamHealth := []cModel.DownstreamItem{}

	var skcDeckAPIDB string

	// get SKC Deck API status by checking the version number. If this operation fails, its save to assume the DB is down.
	if dbVersion, err := skcDeckAPIDBInterface.GetSKCDeckAPIDBVersion(ctx); err != nil {
		downstreamHealth = append(downstreamHealth, cModel.DownstreamItem{ServiceName: "SKC Deck API DB", Status: cModel.Down})
	} else {
		downstreamHealth = append(downstreamHealth, cModel.DownstreamItem{ServiceName: "SKC Deck API DB", Status: cModel.Up})
		skcDeckAPIDB = dbVersion
	}

	status := cModel.APIHealth{Version: "1.0.0", Downstream: downstreamHealth}

	logger.Info("API status check completed", "db_version", skcDeckAPIDB)
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(status)
}

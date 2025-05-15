package downstream

import (
	"context"
	"fmt"
	"net/http"

	cModel "github.com/ygo-skc/skc-go/common/model"
	cUtil "github.com/ygo-skc/skc-go/common/util"
	"github.com/ygo-skc/skc-go/common/ygo"
	"google.golang.org/grpc/status"
)

const (
	BATCH_CARD_INFO_ENDPOINT  = "/api/v1/suggestions/card-details"
	BATCH_CARD_INFO_OPERATION = "Batch Card Info"
	BATCH_CARD_INFO_ERROR     = "There was an error fetching card info"
)

func FetchBatchCardData(ctx context.Context, cardIDs []string) (*cModel.BatchCardData[cModel.CardIDs], *cModel.APIError) {
	logger := cUtil.LoggerFromContext(ctx)
	logger.Info(fmt.Sprintf("Fetching card info for the following IDs: %v", cardIDs))

	var cards *ygo.Cards
	var err error

	if cards, err = cardServiceClient.QueryCards(ctx, &ygo.Resources{IDs: cardIDs}); err != nil {
		logger.Error(
			fmt.Sprintf("There was an issue calling YGO Service. Operation: %s. Code %s. Error: %s",
				BATCH_CARD_INFO_OPERATION,
				status.Code(err),
				err))
		return nil, &cModel.APIError{Message: BATCH_CARD_INFO_ERROR, StatusCode: http.StatusInternalServerError}
	}

	batchCardData := make(cModel.CardDataMap, len(cards.CardInfo))
	for k, v := range cards.CardInfo {
		batchCardData[k] = cModel.YGOCardGRPC{Card: v}
	}
	return &cModel.BatchCardData[cModel.CardIDs]{CardInfo: batchCardData, UnknownResources: cards.UnknownResources}, nil
}

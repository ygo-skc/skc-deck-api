package io

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/ygo-skc/skc-deck-api/downstream"
	"github.com/ygo-skc/skc-deck-api/model"
	"github.com/ygo-skc/skc-deck-api/util"
)

var (
	deckListCardAndQuantityRegex = regexp.MustCompile("[1-3][xX][0-9]{8}")
)

func DeserializeDeckList(ctx context.Context, dl string) (*model.DeckListBreakdown, *model.APIError) {
	var dlb model.DeckListBreakdown
	var cardData *model.BatchCardData[model.CardIDs]
	var err *model.APIError

	if dlb, err = transformDeckListStringToMap(ctx, dl); err != nil {
		return nil, err
	}

	if cardData, err = downstream.FetchBatchCardData(ctx, dlb.CardIDs); err != nil {
		return nil, err
	} else {
		dlb.AllCards = cardData.CardInfo
		dlb.InvalidIDs = cardData.UnknownResources

		dlb.Partition()
		dlb.Sort()
		return &dlb, nil
	}
}

// Transforms decoded deck list into a map that can be parsed easier.
// The map will use the cardID as key and number of copies in the deck as value.
func transformDeckListStringToMap(ctx context.Context, list string) (model.DeckListBreakdown, *model.APIError) {
	logger := util.LoggerFromContext(ctx)
	tokens := deckListCardAndQuantityRegex.FindAllString(list, -1)

	cardCopiesInDeck := map[string]int{}
	cards := []string{}
	for _, token := range tokens {
		splitToken := strings.Split(strings.ToLower(token), "x")
		quantity, _ := strconv.Atoi(splitToken[0])
		cardID := splitToken[1]

		if _, isPresent := cardCopiesInDeck[cardID]; isPresent {
			logger.Info(
				fmt.Sprintf("Deck list contains multiple instances of the same card {%s}.", cardID))
			return model.DeckListBreakdown{}, &model.APIError{
				Message:    "Deck list contains multiple instance of same card. Make sure a cardID appears only once in the deck list.",
				StatusCode: http.StatusBadRequest}
		}
		cardCopiesInDeck[cardID] = quantity
		cards = append(cards, cardID)
	}

	return model.DeckListBreakdown{CardQuantity: cardCopiesInDeck, CardIDs: cards}, nil
}

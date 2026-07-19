package api

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ygo-skc/skc-deck-api/io"
	"github.com/ygo-skc/skc-deck-api/model"
	cModel "github.com/ygo-skc/skc-go/common/v2/model"
	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
)

const (
	retrieveDeckListOp          = "Retrieve Deck List"
	retrieveDeckFeaturingCardOp = "Retrieve Deck Featuring Card"
)

func getDeckListHandler(res http.ResponseWriter, req *http.Request) {
	deckID := chi.URLParam(req, "deckID")

	logger, ctx := cUtil.InitRequest(req.Context(), apiName, retrieveDeckListOp, slog.String("deck_id", deckID))
	logger.Info("Getting content for deck")

	var deckList *model.DeckList
	var err *cModel.APIError
	if deckList, err = skcDeckAPIDBInterface.GetDeckList(ctx, deckID); err != nil {
		err.HandleServerResponse(res)
		return
	}

	decodedListBytes, decodeErr := base64.StdEncoding.DecodeString(deckList.ContentB64)
	if decodeErr != nil {
		logger.Error("Error decoding deck list", "deck_id", deckID, "content_b64", deckList.ContentB64, "err", decodeErr)
		(&cModel.APIError{Message: "Deck list could not be decoded.", StatusCode: http.StatusInternalServerError}).HandleServerResponse(res)
		return
	}
	decodedList := string(decodedListBytes) // decoded string of list contents

	var deckListBreakdown *model.DeckListBreakdown
	if deckListBreakdown, err = io.DeserializeDeckList(ctx, decodedList); err != nil {
		err.HandleServerResponse(res)
		return
	}
	deckList.MainDeck, deckList.ExtraDeck = deckListBreakdown.GetQuantities()

	logger.Info("Successfully retrieved deck list",
		"deck_name", deckList.Name,
		"content_b64", deckList.ContentB64,
		"num_main_deck_cards", deckList.NumMainDeckCards,
		"num_extra_deck_cards", deckList.NumExtraDeckCards,
	)
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(deckList)
}

func getDecksFeaturingCardHandler(res http.ResponseWriter, req *http.Request) {
	cardID := chi.URLParam(req, "cardID")

	logger, ctx := cUtil.InitRequest(req.Context(), apiName, retrieveDeckFeaturingCardOp, slog.String("card_id", cardID))
	logger.Info("Fetching decks that feature card")

	suggestedDecks := model.SuggestedDecks{}

	if featuredIn, err := skcDeckAPIDBInterface.GetDecksThatFeatureCards(ctx, []string{cardID}); err != nil {
		err.HandleServerResponse(res)
		return
	} else {
		suggestedDecks.FeaturedIn = featuredIn
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(suggestedDecks)
}

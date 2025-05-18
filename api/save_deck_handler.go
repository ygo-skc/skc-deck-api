package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ygo-skc/skc-deck-api/io"
	"github.com/ygo-skc/skc-deck-api/model"
	"github.com/ygo-skc/skc-deck-api/validation"
	cModel "github.com/ygo-skc/skc-go/common/model"
	cUtil "github.com/ygo-skc/skc-go/common/util"
)

const (
	submitNewDeckListOp = "Submit New Deck List"
)

func submitNewDeckListHandler(res http.ResponseWriter, req *http.Request) {
	logger, ctx := cUtil.NewRequestSetup(
		cUtil.ContextWithMetadata(context.Background(), apiName, submitNewDeckListOp), submitNewDeckListOp)
	var deckList model.DeckList

	if err := json.NewDecoder(req.Body).Decode(&deckList); err != nil {
		logger.Error("Error occurred while reading submitNewDeckListHandler request body.")
		cModel.HandleServerResponse(cModel.APIError{Message: "Body could not be deserialized.", StatusCode: http.StatusUnprocessableEntity}, res)
		return
	}

	logger, ctx = cUtil.AddAttribute(ctx, slog.String("deckName", deckList.Name))
	logger.Info(fmt.Sprintf("Client attempting to submit new deck with list contents (in base64) {%s}", deckList.ContentB64))

	// object validation
	if err := validation.Validate(deckList); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(err)
		return
	}

	decodedListBytes, _ := base64.StdEncoding.DecodeString(deckList.ContentB64)
	decodedList := string(decodedListBytes) // decoded string of list contents

	var deckListBreakdown model.DeckListBreakdown
	if dlb, err := io.DeserializeDeckList(ctx, decodedList); err != nil {
		err.HandleServerResponse(res)
		return
	} else {
		deckListBreakdown = *dlb
	}

	if err := deckListBreakdown.Validate(ctx); err != nil {
		err.HandleServerResponse(res)
		return
	}

	// Adding new deck list, fully validate before insertion
	deckList.ContentB64 = base64.StdEncoding.EncodeToString([]byte(deckListBreakdown.ListStringCleanup()))
	deckList.UniqueCards = deckListBreakdown.CardIDs
	deckList.NumMainDeckCards = deckListBreakdown.NumMainDeckCards
	deckList.NumExtraDeckCards = deckListBreakdown.NumExtraDeckCards

	if err := skcDeckAPIDBInterface.InsertDeckList(ctx, deckList); err != nil {
		err.HandleServerResponse(res)
	} else {
		json.NewEncoder(res).Encode(cModel.Success{Message: "Successfully inserted new deck list: " + deckList.Name})
	}
}

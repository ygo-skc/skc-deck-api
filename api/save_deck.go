package api

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ygo-skc/skc-deck-api/io"
	"github.com/ygo-skc/skc-deck-api/model"
	"github.com/ygo-skc/skc-deck-api/validation"
)

func submitNewDeckListHandler(res http.ResponseWriter, req *http.Request) {
	var deckList model.DeckList

	if err := json.NewDecoder(req.Body).Decode(&deckList); err != nil {
		log.Println("Error occurred while reading submitNewDeckListHandler request body.")
		model.HandleServerResponse(model.APIError{Message: "Body could not be deserialized.", StatusCode: http.StatusUnprocessableEntity}, res)
		return
	}

	log.Printf("Client attempting to submit new deck with name {%s} and with list contents (in base64) {%s}", deckList.Name, deckList.ContentB64)

	// object validation
	if err := validation.Validate(deckList); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(err)
		return
	}

	decodedListBytes, _ := base64.StdEncoding.DecodeString(deckList.ContentB64)
	decodedList := string(decodedListBytes) // decoded string of list contents

	var deckListBreakdown model.DeckListBreakdown
	if dlb, err := io.DeserializeDeckList(decodedList); err != nil {
		err.HandleServerResponse(res)
		return
	} else {
		deckListBreakdown = *dlb
	}

	if err := deckListBreakdown.Validate(); err != nil {
		err.HandleServerResponse(res)
		return
	}

	// Adding new deck list, fully validate before insertion
	deckList.ContentB64 = base64.StdEncoding.EncodeToString([]byte(deckListBreakdown.ListStringCleanup()))
	deckList.UniqueCards = deckListBreakdown.CardIDs
	deckList.NumMainDeckCards = deckListBreakdown.NumMainDeckCards
	deckList.NumExtraDeckCards = deckListBreakdown.NumExtraDeckCards

	if err := skcDeckAPIDBInterface.InsertDeckList(deckList); err != nil {
		err.HandleServerResponse(res)
	} else {
		json.NewEncoder(res).Encode(model.Success{Message: "Successfully inserted new deck list: " + deckList.Name})
	}
}

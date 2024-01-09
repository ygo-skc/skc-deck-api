package downstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ygo-skc/skc-deck-api/model"
)

const (
	BATCH_CARD_INFO_ENDPOINT  = "/api/v1/suggestions/card-details"
	BATCH_CARD_INFO_OPERATION = "Batch Card Info"
)

func FetchBatchCardInfo(cardIDs []string) (model.CardDataMap, *model.APIError) {
	log.Printf("Fetching card info for the following IDs: %v", cardIDs)

	var resp *http.Response
	var err error
	var cards map[string]model.Card

	reqBody := new(bytes.Buffer)
	json.NewEncoder(reqBody).Encode(cardIDs)

	if resp, err = http.Post(fmt.Sprintf("http://localhost:90%s", BATCH_CARD_INFO_ENDPOINT), "application/json", reqBody); err != nil {
		log.Println("There was an issue calling Suggestion Engine. Operation: . Error: ", BATCH_CARD_INFO_OPERATION, err)
		return nil, &model.APIError{Message: "Error fetching card info", StatusCode: 500}
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&cards); err != nil && err != io.EOF {
		log.Printf("Error occurred while deserializing output from Suggestion Engine. Operation: %s. Error %v", BATCH_CARD_INFO_OPERATION, err)
		return nil, &model.APIError{Message: "Error fetching card info", StatusCode: 500}
	}

	return cards, nil
}

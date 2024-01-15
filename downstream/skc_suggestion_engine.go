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
	BATCH_CARD_INFO_ERROR     = "Error fetching card info"
)

func FetchBatchCardInfo(cardIDs []string) (*model.BatchCardInfo, *model.APIError) {
	log.Printf("Fetching card info for the following IDs: %v", cardIDs)

	var resp *http.Response
	var err error

	reqBody := new(bytes.Buffer)
	json.NewEncoder(reqBody).Encode(model.BatchCardIDs{CardIDs: cardIDs})

	if resp, err = http.Post(fmt.Sprintf("http://localhost:90%s", BATCH_CARD_INFO_ENDPOINT), "application/json", reqBody); err != nil {
		log.Printf("There was an issue calling Suggestion Engine. Operation: %s. Error: %s", BATCH_CARD_INFO_OPERATION, err)
		return nil, &model.APIError{Message: BATCH_CARD_INFO_ERROR, StatusCode: http.StatusInternalServerError}
	} else {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		s, _ := io.ReadAll(resp.Body)
		log.Printf("Suggestion Engine returned with non 200 status. Operation: %s. Body: %s. Code: %d", BATCH_CARD_INFO_OPERATION, string(s), resp.StatusCode)
		return nil, &model.APIError{Message: BATCH_CARD_INFO_ERROR, StatusCode: http.StatusInternalServerError}
	}

	var cardData model.BatchCardInfo
	if err = json.NewDecoder(resp.Body).Decode(&cardData); err != nil && err != io.EOF {
		log.Printf("Error occurred while deserializing output from Suggestion Engine. Operation: %s. Error %v", BATCH_CARD_INFO_OPERATION, err)
		return nil, &model.APIError{Message: "Error fetching card info", StatusCode: http.StatusInternalServerError}
	}

	return &cardData, nil
}
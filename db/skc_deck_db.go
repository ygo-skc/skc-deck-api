package db

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ygo-skc/skc-deck-api/model"
	cModel "github.com/ygo-skc/skc-go/common/v3/model"
	cUtil "github.com/ygo-skc/skc-go/common/v3/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	skcDeckDB          *mongo.Database
	deckListCollection *mongo.Collection
)

// interface
type SKCDeckAPIDAO interface {
	GetSKCDeckAPIDBVersion(context.Context) (string, error)

	InsertDeckList(context.Context, model.DeckList) *cModel.APIError
	GetDeckList(context.Context, string) (*model.DeckList, *cModel.APIError)
	GetDecksThatFeatureCards(context.Context, []string) ([]model.DeckList, *cModel.APIError)
}

// impl
type SKCDeckAPIDAOImplementation struct{}

// Retrieves the version number of the SKC Deck API DB or throws an error if an exception occurs.
func (dbInterface SKCDeckAPIDAOImplementation) GetSKCDeckAPIDBVersion(ctx context.Context) (string, error) {
	logger := cUtil.RetrieveLogger(ctx)

	var commandResult bson.M
	command := bson.D{{Key: "serverStatus", Value: 1}}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := skcDeckDB.RunCommand(ctx, command).Decode(&commandResult); err != nil {
		logger.Info("Error getting SKC Deck API DB version", "err", err)
		return "", err
	} else {
		return fmt.Sprintf("%v", commandResult["version"]), nil
	}
}

func (dbInterface SKCDeckAPIDAOImplementation) InsertDeckList(ctx context.Context,
	deckList model.DeckList) *cModel.APIError {
	logger := cUtil.RetrieveLogger(ctx)

	deckList.CreatedAt = time.Now()
	deckList.UpdatedAt = deckList.CreatedAt

	logger.Info("Inserting deck list",
		"deck_name", deckList.Name,
		"num_main_deck_cards", deckList.NumMainDeckCards,
		"num_extra_deck_cards", deckList.NumExtraDeckCards,
		"content_b64", deckList.ContentB64,
	)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if res, err := deckListCollection.InsertOne(ctx, deckList); err != nil {
		logger.Error("Error saving new deck list into DB", "err", err)
		return &cModel.APIError{Message: "There was a problem saving deck list", StatusCode: http.StatusInternalServerError}
	} else {
		logger.Info("Successfully inserted new deck list into DB", "deck_id", res.InsertedID)
		return nil
	}
}

func (dbInterface SKCDeckAPIDAOImplementation) GetDeckList(ctx context.Context, deckID string) (*model.DeckList, *cModel.APIError) {
	logger := cUtil.RetrieveLogger(ctx)

	if objectId, err := bson.ObjectIDFromHex(deckID); err != nil {
		logger.Error("Error retrieving deck from DB - invalid deck ID", "deck_id", deckID)
		return nil, &cModel.APIError{Message: "Deck ID not valid", StatusCode: http.StatusBadRequest}
	} else {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		var dl model.DeckList
		if err := deckListCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&dl); err != nil {
			logger.Error("Error retrieving deck from DB", "deck_id", deckID, "err", err)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, &cModel.APIError{Message: "Deck w/ ID not found", StatusCode: http.StatusNotFound}
			} else {
				return nil, &cModel.APIError{Message: "Error retrieving deck", StatusCode: http.StatusInternalServerError}
			}
		} else {
			return &dl, nil
		}
	}
}

func (dbInterface SKCDeckAPIDAOImplementation) GetDecksThatFeatureCards(ctx context.Context,
	cardIDs []string) ([]model.DeckList, *cModel.APIError) {
	logger := cUtil.RetrieveLogger(ctx)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// select only these fields from collection
	opts := options.Find().SetProjection(
		bson.D{
			{Key: "name", Value: 1}, {Key: "videoUrl", Value: 1}, {Key: "uniqueCards", Value: 1}, {Key: "deckMascots", Value: 1}, {Key: "numMainDeckCards", Value: 1},
			{Key: "numExtraDeckCards", Value: 1}, {Key: "tags", Value: 1}, {Key: "createdAt", Value: 1}, {Key: "updatedAt", Value: 1},
		},
	)

	if cursor, err := deckListCollection.Find(ctx, bson.M{"uniqueCards": bson.M{"$in": cardIDs}}, opts); err != nil {
		logger.Error("Error retrieving all deck lists that feature cards", "card_ids", cardIDs, "err", err)
		return nil, &cModel.APIError{Message: "Error retrieving deck suggestions", StatusCode: http.StatusInternalServerError}
	} else {
		dl := []model.DeckList{}
		if err := cursor.All(ctx, &dl); err != nil {
			logger.Error("Error retrieving all deck lists that feature cards", "card_ids", cardIDs, "err", err)
			return nil, &cModel.APIError{Message: "Error retrieving deck suggestions", StatusCode: http.StatusInternalServerError}
		}

		return dl, nil
	}
}

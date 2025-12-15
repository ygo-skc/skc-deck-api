package db

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ygo-skc/skc-deck-api/model"
	cModel "github.com/ygo-skc/skc-go/common/v2/model"
	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
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
	GetDecksThatFeatureCards(context.Context, []string) (*[]model.DeckList, *cModel.APIError)
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
		logger.Info(fmt.Sprintf("Error getting SKC Deck API DB version %v", err))
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

	logger.Info(
		fmt.Sprintf("Inserting deck with name %s with Main Deck size %d and Extra Deck size %d. List contents (in base64 and possibly reformatted) %s",
			deckList.Name, deckList.NumMainDeckCards, deckList.NumExtraDeckCards, deckList.ContentB64))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if res, err := deckListCollection.InsertOne(ctx, deckList); err != nil {
		logger.Error(fmt.Sprintf("Error saving new deck list into DB, error: %v", err))
		return &cModel.APIError{Message: "There was a problem saving deck list", StatusCode: http.StatusInternalServerError}
	} else {
		logger.Info(fmt.Sprintf("Successfully inserted new deck list into DB, deck ID: %s", res.InsertedID))
		return nil
	}
}

func (dbInterface SKCDeckAPIDAOImplementation) GetDeckList(ctx context.Context, deckID string) (*model.DeckList, *cModel.APIError) {
	logger := cUtil.RetrieveLogger(ctx)

	if objectId, err := bson.ObjectIDFromHex(deckID); err != nil {
		logger.Error("Error retrieving deck from DB - nvalid deck ID")
		return nil, &cModel.APIError{Message: "Deck ID not valid", StatusCode: http.StatusBadRequest}
	} else {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		var dl model.DeckList
		if err := deckListCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&dl); err != nil {
			logger.Error(fmt.Sprintf("Error retrieving deck from DB. Err: %v", err))
			if err.Error() == "mongo: no documents in result" {
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
	cardIDs []string) (*[]model.DeckList, *cModel.APIError) {
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
		logger.Error(fmt.Sprintf("Error retrieving all deck lists that feature cards w/ ID %v. Err: %v", cardIDs, err))
		return nil, &cModel.APIError{Message: "Error retrieving deck suggestions", StatusCode: http.StatusInternalServerError}
	} else {
		dl := []model.DeckList{}
		if err := cursor.All(ctx, &dl); err != nil {
			logger.Error(fmt.Sprintf("Error retrieving all deck lists that feature cards w/ ID %v. Err: %v", cardIDs, err))
			return nil, &cModel.APIError{Message: "Error retrieving deck suggestions", StatusCode: http.StatusInternalServerError}
		}

		return &dl, nil
	}
}

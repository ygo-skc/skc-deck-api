package model

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	cModel "github.com/ygo-skc/skc-go/common/v2/model"
	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SuggestedDecks struct {
	FeaturedIn *[]DeckList `json:"featuredIn"`
}

type DeckList struct {
	ID                bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name              string         `bson:"name" json:"name" validate:"required,decklistname"`
	ContentB64        string         `bson:"content" json:"content" validate:"required,base64"`
	VideoUrl          string         `bson:"videoUrl" json:"videoUrl" validate:"omitempty,url"`
	UniqueCards       cModel.CardIDs `bson:"uniqueCards" json:"uniqueCards" validate:"omitempty"`
	DeckMascots       cModel.CardIDs `bson:"deckMascots" json:"deckMascots" validate:"omitempty,deckmascots"`
	NumMainDeckCards  int            `bson:"numMainDeckCards" json:"numMainDeckCards"`
	NumExtraDeckCards int            `bson:"numExtraDeckCards" json:"numExtraDeckCards"`
	Tags              []string       `bson:"tags" json:"tags" validate:"required"`
	CreatedAt         time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time      `bson:"updatedAt" json:"updatedAt"`
	MainDeck          []Content      `bson:"mainDeck,omitempty" json:"mainDeck,omitempty"`
	ExtraDeck         []Content      `bson:"extraDeck,omitempty" json:"extraDeck,omitempty"`
}

type Content struct {
	Quantity int            `bson:"omitempty" json:"quantity"`
	Card     cModel.YGOCard `bson:"omitempty" json:"card"`
}

type DeckListBreakdown struct {
	CardQuantity      map[string]int
	CardIDs           cModel.CardIDs
	InvalidIDs        cModel.CardIDs
	AllCards          cModel.CardDataMap
	MainDeck          cModel.YGOCards
	ExtraDeck         cModel.YGOCards
	NumMainDeckCards  int
	NumExtraDeckCards int
}

func (dlb *DeckListBreakdown) Partition() {
	dlb.MainDeck, dlb.ExtraDeck = []cModel.YGOCard{}, []cModel.YGOCard{}
	dlb.NumMainDeckCards, dlb.NumExtraDeckCards = 0, 0

	for _, cardID := range dlb.CardIDs {
		if _, isPresent := dlb.AllCards[cardID]; isPresent {
			if cModel.IsExtraDeckMonster(dlb.AllCards[cardID]) {
				dlb.ExtraDeck = append(dlb.ExtraDeck, dlb.AllCards[cardID])
				dlb.NumExtraDeckCards += dlb.CardQuantity[cardID]
			} else {
				dlb.MainDeck = append(dlb.MainDeck, dlb.AllCards[cardID])
				dlb.NumMainDeckCards += dlb.CardQuantity[cardID]
			}
		}
	}
}

func (dlb *DeckListBreakdown) Sort() {
	dlb.MainDeck.SortCardsByName()
	dlb.ExtraDeck.SortCardsByName()
}

func (dlb *DeckListBreakdown) GetQuantities() ([]Content, []Content) {
	mainDeckContent := make([]Content, 0, len(dlb.MainDeck))
	for _, card := range dlb.MainDeck {
		mainDeckContent = append(mainDeckContent, Content{Card: card, Quantity: dlb.CardQuantity[card.GetID()]})
	}

	extraDeckContent := make([]Content, 0, len(dlb.ExtraDeck))
	for _, card := range dlb.ExtraDeck {
		extraDeckContent = append(extraDeckContent, Content{Card: card, Quantity: dlb.CardQuantity[card.GetID()]})
	}

	return mainDeckContent, extraDeckContent
}

func (dlb DeckListBreakdown) ListStringCleanup() string {
	var formattedDLS strings.Builder
	formattedDLS.WriteString("Main Deck\n")

	for _, card := range dlb.MainDeck {
		formattedDLS.WriteString(formattedLine(card, dlb.CardQuantity[card.GetID()]))
	}

	formattedDLS.WriteString("\nExtra Deck\n")

	for _, card := range dlb.ExtraDeck {
		formattedDLS.WriteString(formattedLine(card, dlb.CardQuantity[card.GetID()]))
	}

	return formattedDLS.String()
}

func formattedLine(card cModel.YGOCard, quantity int) string {
	return fmt.Sprintf("%dx%s|%s\n", quantity, card.GetID(), card.GetName())
}

func (dlb DeckListBreakdown) Validate(ctx context.Context) *cModel.APIError {
	var msg = ""

	if len(dlb.InvalidIDs) > 0 {
		msg = fmt.Sprintf("Deck list contains card(s) that were not found in skc DB. List of Card ID's not found: %v", dlb.InvalidIDs)
	} else if dlb.NumExtraDeckCards > 15 { // validate extra deck has correct number of cards
		msg = fmt.Sprintf("Extra deck cannot contain more than 15 cards. Current deck contains %d extra deck cards.", dlb.NumExtraDeckCards)
	} else if dlb.NumMainDeckCards < 40 || dlb.NumMainDeckCards > 60 { // validate main deck has correct number of cards
		msg = fmt.Sprintf("Main deck cannot contain less than 40 cards and no more than 60 cards. Current deck contains %d main deck cards.",
			dlb.NumMainDeckCards)
	}

	if msg != "" {
		cUtil.RetrieveLogger(ctx).Error(msg)
		return &cModel.APIError{Message: msg, StatusCode: http.StatusBadRequest}
	} else {
		return nil
	}
}

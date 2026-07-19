package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

const (
	maxPoolSize = 30
)

// connects to Deck API database
func EstablishSKCDeckAPIDBConn() {
	certificateKeyFilePath := "./certs/skc-deck-api-db.pem"
	uri := fmt.Sprintf("%s/?tlsCertificateKeyFile=%s", cUtil.EnvMap["DB_HOST"], certificateKeyFilePath)

	credential := options.Credential{
		AuthMechanism: "MONGODB-X509",
	}

	if client, err := mongo.Connect(options.Client().
		ApplyURI(uri).
		SetAuth(credential).
		SetMaxPoolSize(maxPoolSize).
		SetMaxConnIdleTime(20 * time.Minute).
		SetTimeout(2 * time.Second).
		SetReadConcern(readconcern.Majority()).   // prefer strongly consistent reeds
		SetWriteConcern(writeconcern.Majority()). // writes to most replicas before acknowledging the write is complete
		SetCompressors([]string{"zlib"}).
		SetAppName("SKC Deck API")); err != nil {
		slog.Error("Error creating new mongodb client for skc-deck-api-db", "err", err)
		os.Exit(1)
	} else {
		skcDeckDB = client.Database("deckDB")
	}

	// init collections
	deckListCollection = skcDeckDB.Collection("lists")

	if err := createIndexes(); err != nil {
		slog.Error("Error creating indexes for skc-deck-api-db", "err", err)
		os.Exit(1)
	}

	slog.Info("Connected to deck DB")
}

func createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := deckListCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "uniqueCards", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "uniqueCards", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return err
}

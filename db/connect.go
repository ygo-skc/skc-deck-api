package db

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	cUtil "github.com/ygo-skc/skc-go/common/v2/util"
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
		log.Fatalln("Error creating new mongodb client for skc-deck-api-db", err)
	} else {
		skcDeckDB = client.Database("deckDB")
	}

	// init collections
	deckListCollection = skcDeckDB.Collection("lists")

	slog.Info("Connected to deck DB")
}

package helpers

import (
	"context"

	"codebase/models"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type (
	Mongo interface {
		InsertOne(document any, ctx context.Context) (*string, error)
	}

	MongoImpl struct {
		shareHolder *models.SharedHolder
	}
)

func NewMongo(sharedHolder *models.SharedHolder) Mongo {
	return &MongoImpl{sharedHolder}
}

func (m *MongoImpl) InsertOne(document any, ctx context.Context) (*string, error) {
	collection := m.shareHolder.Mongo.Database(m.shareHolder.Config.MongoDb).Collection(m.shareHolder.Config.MongoCollection)

	result, err := collection.InsertOne(ctx, document)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	insertedId := result.InsertedID.(primitive.ObjectID).Hex()

	return &insertedId, nil
}

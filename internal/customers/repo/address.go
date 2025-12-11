package repo

import (
	"context"
	"fmt"
	"log/slog"

	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type customersAddressRepo struct {
	db    *bun.DB
	mongo *mongo.Client
}

func NewCustomersAddressRepo(db *bun.DB, mongo *mongo.Client) ports.CustomersAddressRepo {
	return &customersAddressRepo{db, mongo}
}

func (c *customersAddressRepo) RegisterAddress(ctx context.Context, address models.Address) error {

	coll := c.mongo.Database("customers").Collection("addresses")

	_, err := coll.InsertOne(ctx, address)
	if err != nil {
		slog.Error("Error inserting address into MongoDB", "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (c *customersAddressRepo) SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (models.Addresses, error) {

	var addresses models.Addresses

	coll := c.mongo.Database("customers").Collection("addresses")
	cursor, err := coll.Find(ctx, bson.M{"customer_id": customer_id.String()})
	if err != nil {
		slog.Error("Error finding addresses in MongoDB", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &addresses); err != nil {
		slog.Error("Error decoding addresses from MongoDB cursor", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if len(addresses) == 0 {
		return nil, errcustom.ErrNotFound
	}

	return addresses, nil

}

func (c *customersAddressRepo) DeleteAddress(ctx context.Context, customer_id uuid.UUID, address_id string) error {

	id, err := primitive.ObjectIDFromHex(address_id)
	if err != nil {
		slog.Error("Error converting address ID to ObjectID", "error", err)
		return errcustom.ErrUnexpectedError
	}

	coll := c.mongo.Database("customers").Collection("addresses")

	result, err := coll.DeleteOne(ctx, bson.M{"_id": id, "customer_id": customer_id.String()})
	if err != nil {
		slog.Error("Error deleting address from MongoDB", "error", err)
		return errcustom.ErrUnexpectedError

	}

	fmt.Println(result)

	return nil
}

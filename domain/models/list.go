package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type List struct {
	ID       primitive.ObjectID `bson:"_id"`
	Username string             `bson:"username"`
	Count    uint64             `bson:"count"`
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/LWich/git-views/config"
	"github.com/LWich/git-views/domain/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrOnlyAcceptedReadme  = errors.New("err request only accepted from github readme")
	ErrLenOfNameCantBeZero = errors.New("err len of name can`t be zero")
)

func main() {
	mongoUri, port := initMongoUriAndPort()

	ctx := context.TODO()

	db, err := newDb(ctx, mongoUri)
	if err != nil {
		log.Fatal(err)
	}

	listCollection := db.Collection(listCollectionName)

	http.Handle("/", checkGitCamo(handleViews(ctx, listCollection)))

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func initMongoUriAndPort() (mongoUri, port string) {
	if !config.USE_HEROKU {
		mongoUri = config.MONGODB_URI
		port = config.PORT
	} else {
		port = ":" + os.Getenv("PORT")
		mongoUri = os.Getenv("MONGODB_URI")
	}

	return mongoUri, port
}

func checkGitCamo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(req.UserAgent()) != 0 {
			if !strings.Contains(req.UserAgent(), "github-camo") {
				http.Error(w, ErrOnlyAcceptedReadme.Error(), http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, req)
	})
}

func handleViews(ctx context.Context, collection *mongo.Collection) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := req.URL.Query().Get("username")

		if len(username) == 0 {
			http.Error(w, ErrLenOfNameCantBeZero.Error(), http.StatusBadRequest)
			return
		}

		var result models.List

		if err := collection.FindOne(
			ctx,
			bson.M{"username": username},
		).Decode(&result); err != nil {
			if err == mongo.ErrNoDocuments {
				newDocument := models.List{
					ID:       primitive.NewObjectID(),
					Username: username,
					Count:    1,
				}

				if _, err := collection.InsertOne(ctx, newDocument); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				result = newDocument

				svg_image(w, 0)
				return
			}
		}

		collection.FindOneAndUpdate(ctx, bson.M{"_id": result.ID}, bson.M{"$set": bson.M{"count": result.Count + 1}})

		svg_image(w, result.Count)
	}
}

func newDb(ctx context.Context, databaseUrl string) (*mongo.Database, error) {
	clientOptions := options.Client().ApplyURI(databaseUrl)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client.Database("users"), nil
}

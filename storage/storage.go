package storage

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/sattellite/tg-group-control-bot/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Storage contains database connection
type Storage struct {
	Client *mongo.Client
	Name   string
}

// New creates connection to storage
func New(cfg *config.Config, db string) (*Storage, error) {
	var s Storage
	s.Name = db
	client, err := mongo.NewClient(options.Client().ApplyURI(cfg.MongoURL))

	ctx, cancelConnectCtx := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnectCtx()
	err = client.Connect(ctx)
	if err != nil {
		s.Client = client
		return &s, err
	}

	ctx, cancelCheckCtx := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCheckCtx()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return &s, errors.New("Failed ping in New:" + err.Error())
	}

	s.Client = client
	return &s, err
}

func (s *Storage) checkDB() (context.Context, context.CancelFunc, error) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 2*time.Second)
	err := s.Client.Ping(ctx, readpref.Primary())
	return ctx, cancelCtx, err
}

// CheckUser is checking user for ban and existence or creating a new user
func (s *Storage) CheckUser(u config.User) (bool, config.User, error) {
	var result config.User
	var isNewUser bool

	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return false, result, errors.New("Failed ping in CheckUser:" + err.Error())
	}

	collection := s.Client.Database(s.Name).Collection("users")

	// Trying find user
	err = collection.FindOne(ctx, bson.M{"ID": u.ID}).Decode(&result)
	if err != nil {
		// Replace zero userID to passed
		result.ID = u.ID
		if err != mongo.ErrNoDocuments {
			return false, result, errors.New("Failed find ID in CheckUser:" + err.Error())
		}
		// User doesn't exist
		isNewUser = true
	}

	if result.Banned {
		return false, result, errors.New("This user ID " + strconv.Itoa(u.ID) + " was banned at " + strconv.FormatInt(result.BanDate, 10))
	}

	// Create new user
	if isNewUser {
		unixtime := time.Now().Unix()
		u.RegDate = unixtime
		u.UsageDate = unixtime
		_, err := collection.InsertOne(ctx, u)
		if err != nil {
			return isNewUser, result, errors.New("Failed insert in CheckUser:" + err.Error())
		}
	}

	// Update activity time of exists user
	if !isNewUser {
		usageDate := time.Now().Unix()
		// Renewing usagedate, firstname, lastname, username
		_, err := collection.UpdateOne(ctx, bson.M{"ID": u.ID}, bson.M{"$set": bson.M{
			"UsageDate": usageDate,
			"FirstName": u.FirstName,
			"LastName":  u.LastName,
			"UserName":  u.UserName,
		}})
		if err != nil {
			return isNewUser, result, errors.New("Failed update in CheckUser: " + err.Error())
		}
		result.UsageDate = usageDate
	}

	return isNewUser, result, nil
}

// UpdateUser updates user's passed fields
func (s *Storage) UpdateUser(u config.User) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.New("Failed ping in UpdateUser:" + err.Error())
	}

	collection := s.Client.Database(s.Name).Collection("users")

	usageDate := time.Now().Unix()
	u.UsageDate = usageDate
	_, err = collection.UpdateOne(ctx, bson.M{"ID": u.ID}, bson.M{"$set": u})
	if err != nil {
		return errors.New("Failed update in UpdateUser: " + err.Error())
	}
	return nil
}

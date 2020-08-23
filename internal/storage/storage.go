package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"tg-group-control-bot/internal/config"

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
		return &s, errors.Wrap(err, "Failed ping in New")
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
		return false, result, errors.Wrap(err, "Failed ping in CheckUser")
	}

	collection := s.Client.Database(s.Name).Collection("users")

	// Trying find user
	err = collection.FindOne(ctx, bson.M{"ID": u.ID}).Decode(&result)
	if err != nil {
		// Replace zero userID to passed
		result.ID = u.ID
		if err != mongo.ErrNoDocuments {
			return false, result, errors.Wrap(err, "Failed find ID in CheckUser")
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
		u.Chats = make([]int64, 0)
		_, err := collection.InsertOne(ctx, u)
		if err != nil {
			return isNewUser, result, errors.Wrap(err, "Failed insert in CheckUser")
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
			return isNewUser, result, errors.Wrap(err, "Failed update in CheckUser")
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
		return errors.Wrap(err, "Failed ping in UpdateUser")
	}

	collection := s.Client.Database(s.Name).Collection("users")

	usageDate := time.Now().Unix()
	u.UsageDate = usageDate
	_, err = collection.UpdateOne(ctx, bson.M{"ID": u.ID}, bson.M{"$set": u})
	if err != nil {
		return errors.Wrap(err, "Failed update in UpdateUser")
	}
	return nil
}

// UpdateChat updates chat information
func (s *Storage) UpdateChat(chat config.Chat) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in UpdateChat")
	}

	collection := s.Client.Database(s.Name).Collection("chats")

	// _, err = collection.UpdateOne(ctx, bson.M{"ID": chat.ID}, bson.M{"$set": chat}, options.Update().SetUpsert(true))
	var c config.Chat
	var needCreate bool
	err = collection.FindOne(ctx, bson.M{"ID": chat.ID}).Decode(&c)
	if err != nil {
		if err.Error() != mongo.ErrNoDocuments.Error() {
			return errors.Wrap(err, "Failed find in UpdateChat")
		}
		needCreate = true
	}

	if needCreate {
		_, err = collection.InsertOne(ctx, chat)
		if err != nil {
			return errors.Wrap(err, "Failed insert in UpdateChat")
		}
		return nil
	}

	c.Title = chat.Title
	_, err = collection.UpdateOne(ctx, bson.M{"ID": chat.ID}, bson.M{"$set": c})
	if err != nil {
		return errors.Wrap(err, "Failed update in UpdateChat")
	}
	return nil
}

// UserConfirmed checks chat for user confirmation
func (s *Storage) UserConfirmed(chatID int64, userID int) (bool, error) {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return true, errors.Wrap(err, "Failed ping in UserConfirmed")
	}

	var c config.Chat
	collection := s.Client.Database(s.Name).Collection("chats")
	err = collection.FindOne(ctx, bson.M{"ID": chatID}, options.FindOne().SetProjection(bson.M{
		"_id": 0,
		"Users": bson.M{
			"$elemMatch": bson.M{"ID": userID},
		},
	})).Decode(&c)

	if len(c.Users) == 0 {
		return false, err
	}

	return c.Users[0].Confirmed, err
}

// AddChatUser adding user to passed chat
func (s *Storage) AddChatUser(chatID int64, cu config.ChatUser) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in AddChatUser")
	}

	collection := s.Client.Database(s.Name).Collection("chats")
	_, err = collection.UpdateOne(ctx, bson.M{"ID": chatID}, bson.M{"$push": bson.M{"Users": cu}})

	return err
}

// UpdateConfirmReference set reference to confirmation message in chat
func (s *Storage) UpdateConfirmReference(chatID int64, msgID, userID int) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in AddChatUser")
	}

	collection := s.Client.Database(s.Name).Collection("chats")

	_, err = collection.UpdateOne(ctx, bson.M{"ID": chatID, "Users.ID": userID}, bson.M{"$set": bson.M{"Users.$.ConfirmMsg": config.Ref{
		ChatID: chatID,
		MsgID:  msgID,
	}}})

	return err
}

// GetChatInfo returns chat info
func (s *Storage) GetChatInfo(chatID int64) (config.Chat, error) {
	var c config.Chat
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return c, errors.Wrap(err, "Failed ping in GetChatInfo")
	}

	collection := s.Client.Database(s.Name).Collection("chats")
	err = collection.FindOne(ctx, bson.M{"ID": chatID}).Decode(&c)
	return c, err
}

// GetChatAdmins return list of chat admins
func (s *Storage) GetChatAdmins(chatID int64) []int {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return []int{}
	}

	var c config.Chat
	collection := s.Client.Database(s.Name).Collection("chats")
	err = collection.FindOne(ctx, bson.M{"ID": chatID}).Decode(&c)
	return c.Admins
}

// GetChatTitle return title of the chat
func (s *Storage) GetChatTitle(chatID int64) string {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return ""
	}

	var c config.Chat
	collection := s.Client.Database(s.Name).Collection("chats")
	err = collection.FindOne(ctx, bson.M{"ID": chatID}).Decode(&c)

	if c.Type == "supergroup" {
		return "@" + c.UserName
	}
	return c.Title
}

// RemoveUnconfirmedChatUser removes unconfirmed user from chat and returns reference to confirm message
func (s *Storage) RemoveUnconfirmedChatUser(chatID int64, userID int) (config.Ref, error) {
	var ref config.Ref
	var c config.Chat
	isNeedRemove := false

	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return ref, errors.Wrap(err, "Failed ping in RemoveUnconfirmedChatUser")
	}
	collection := s.Client.Database(s.Name).Collection("chats")

	// Find unconfirmed user
	err = collection.FindOne(ctx, bson.M{"ID": chatID}, options.FindOne().SetProjection(bson.M{
		"_id": 0,
		"Users": bson.M{
			"$elemMatch": bson.M{"ID": userID, "Confirmed": false},
		},
	})).Decode(&c)

	if err != nil {
		return ref, err
	}

	if len(c.Users) > 0 {
		ref = c.Users[0].ConfirmMsg
		isNeedRemove = true
	}

	if isNeedRemove {
		_, err = collection.UpdateOne(ctx, bson.M{"ID": chatID}, bson.M{
			"$pull": bson.M{
				"Users": bson.M{"ID": userID, "Confirmed": false},
			},
		})
		if err != nil {
			return ref, errors.Wrap(err, "Failed remove user in RemoveUnconfirmedChatUser")
		}
	}

	return ref, err
}

// ConfirmChatUser set user confirmed and returns reference to confirm message
func (s *Storage) ConfirmChatUser(chatID int64, userID int) (config.Ref, error) {
	var ref config.Ref
	var c config.Chat
	isNeedConfirm := false

	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return ref, errors.Wrap(err, "Failed ping in ConfirmChatUser")
	}
	collection := s.Client.Database(s.Name).Collection("chats")

	// Find unconfirmed user
	err = collection.FindOne(ctx, bson.M{"ID": chatID}, options.FindOne().SetProjection(bson.M{
		"_id": 0,
		"Users": bson.M{
			"$elemMatch": bson.M{"ID": userID, "Confirmed": false},
		},
	})).Decode(&c)

	if err != nil {
		return ref, err
	}

	if len(c.Users) > 0 {
		ref = c.Users[0].ConfirmMsg
		isNeedConfirm = true
	}

	if isNeedConfirm {
		_, err = collection.UpdateOne(ctx, bson.M{"ID": chatID, "Users.ID": userID}, bson.M{
			"$set": bson.M{
				"Users.$.Confirmed":  true,
				"Users.$.ConfirmMsg": bson.M{"ChatID": 0, "MsgID": 0},
			},
		})
		if err != nil {
			return ref, errors.Wrap(err, "Failed update user in ConfirmChatUser")
		}
	}

	return ref, err
}

// RemoveChatAdmin removes user from admins list
func (s *Storage) RemoveChatAdmin(chatID int64, userID int) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in RemoveChatAdmin")
	}
	collection := s.Client.Database(s.Name).Collection("chats")
	_, err = collection.UpdateOne(ctx, bson.M{"ID": chatID}, bson.M{
		"$pull": bson.M{"Admins": userID},
	})
	if err != nil {
		return errors.Wrap(err, "Failed remove user in RemoveUnconfirmedChatUser")
	}
	return nil
}

// AddUnconfirmedChat adding chat to user's unconfirmed chats
func (s *Storage) AddUnconfirmedChat(chatID int64, userID int) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in AddUnconfirmedChat")
	}

	collection := s.Client.Database(s.Name).Collection("users")
	_, err = collection.UpdateOne(ctx, bson.M{"ID": userID}, bson.M{"$push": bson.M{"Chats": chatID}})

	return err
}

// DeleteUnconfirmedChat from user's unconfirmed chats
func (s *Storage) DeleteUnconfirmedChat(chatID int64, userID int) error {
	ctx, cancelCtx, err := s.checkDB()
	defer cancelCtx()
	if err != nil {
		return errors.Wrap(err, "Failed ping in DeleteUnconfirmedChat")
	}

	collection := s.Client.Database(s.Name).Collection("users")
	_, err = collection.UpdateOne(ctx, bson.M{"ID": userID}, bson.M{"$pull": bson.M{"Chats": chatID}})

	return err
}

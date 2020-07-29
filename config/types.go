package config

// Config is main application configuration struct
type Config struct {
	Debug         bool   `env:"DEBUG" envDefault:"false"`
	TelegramDebug bool   `env:"TG_DEBUG" envDefault:"false"`
	BotToken      string `env:"BOT_TOKEN,required"`
	MongoURL      string `env:"MONGO_URL,required"`
}

// User describes all meta data
type User struct {
	ID        int    `json:"ID" bson:"ID"`
	FirstName string `json:"FirstName" bson:"FirstName"`
	LastName  string `json:"LastName" bson:"LastName"`
	UserName  string `json:"UserName" bson:"UserName"`
	Language  string `json:"Language" bson:"Language"`
	Bot       bool   `json:"Bot" bson:"Bot"`
	Banned    bool   `json:"Banned" bson:"Banned"`
	BanDate   int64  `json:"BanDate" bson:"BanDate"`
	RegDate   int64  `json:"RegDate" bson:"RegDate"`
	UsageDate int64  `json:"UsageDate" bson:"UsageDate"`
}

// String displays a simple text version of a user.
// It is normally a user's username, but falls back to a first/last
// name as available.
func (u *User) String() string {
	if u.UserName != "" {
		return u.UserName
	}

	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}

	return name
}

// Chat describes chat where bot placed
type Chat struct {
	ID     int64      `json:"ID" bson:"ID"`
	Users  []ChatUser `json:"Users" bson:"Users"`
	Title  string     `json:"Title" bson:"Title"`
	Type   string     `json:"Type" bson:"Type"`
	Admins []int      `json:"Admins" bson:"Admins"`
}

// ChatUser describes user in chat
type ChatUser struct {
	ID         int    `json:"ID" bson:"ID"`
	Confirmed  bool   `json:"Confirmed" bson:"Confirmed"`
	ConfirmMsg Ref    `json:"ConfirmMsg" bson:"ConfirmMsg"`
	MsgCount   uint64 `json:"MsgCount" bson:"MsgCount"`
}

// Ref describe messages in chats
type Ref struct {
	ChatID int64 `json:"ChatID" bson:"ChatID"`
	MsgID  int   `json:"MsgID" bson:"MsgID"`
}

package database

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
	Content   string    `gorm:"type:text" json:"content"`
	Iv        string    `gorm:"type:varchar(25)" json:"iv"`
	ChatID    string    `gorm:"type:varchar(36)" json:"chatId"`
	SenderID  string    `gorm:"type:varchar(36)" json:"senderId"`
	Chat      Chat      `gorm:"foreignKey:ChatID" json:"-"`
	Sender    User      `gorm:"foreignKey:SenderID" json:"-"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.NewString()
	return
}

type Chat struct {
	ID          string       `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time    `gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time    `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	Name        string       `gorm:"type:varchar(255)"`
	Picture     *string      `gorm:"type:varchar(2048)"` // TODO: adjust for s3 file key
	Description *string      `gorm:"type:text"`
	Messages    []Message    `gorm:"foreignKey:ChatID"`
	Users       []User       `gorm:"many2many:chats_users;"`
	ChatsUsers  []ChatsUsers `gorm:"foreignKey:ChatID"`
}

func (c *Chat) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.NewString()
	return
}

type User struct {
	ID               string       `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedAt        time.Time    `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt        time.Time    `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
	Email            string       `gorm:"type:varchar(255);uniqueIndex" json:"email"`
	Password         string       `gorm:"type:text" json:"-"`
	Name             string       `gorm:"type:varchar(50)" json:"name"`
	Bio              *string      `gorm:"type:varchar(1000)" json:"bio"`
	Iv               string       `gorm:"type:varchar(16)" json:"iv"`
	WrapingKeyRandom string       `gorm:"type:varchar(16)" json:"wrapingKeyRandom"`
	ProfilePicture   *string      `gorm:"type:varchar(512)" json:"profilePicture"`
	PublicKey        string       `gorm:"type:text" json:"publicKey"`
	PrivateKey       string       `gorm:"type:text" json:"privateKey"`
	Chats            []Chat       `gorm:"many2many:chats_users;"`
	ChatsUsers       []ChatsUsers `gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.NewString()
	buf := make([]byte, 12)
	_, err = rand.Read(buf)
	if err != nil {
		panic("Couldn't create random bytes")
	}
	u.WrapingKeyRandom = base64.RawStdEncoding.EncodeToString(buf)
	return
}

type ChatsUsers struct {
	ChatID    string    `gorm:"type:varchar(36);primaryKey"`
	UserID    string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	Key       string    `gorm:"type:text"`
	Chat      Chat      `gorm:"foreignKey:ChatID"`
	User      User      `gorm:"foreignKey:UserID"`
}

type UserKeys struct {
	UserID    string    `gorm:"type:varchar(36);primaryKey"`
	Random    string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	ExpiredAt time.Time `gorm:"type:datetime"`
	User      User      `gorm:"foreignKey:UserID"`
}

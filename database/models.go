package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string `gorm:"primary_key;type:string;size:36"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	BaseModel          `gorm:"embedded"`
	FullName           string `gorm:"size:255"`
	UserName           string `gorm:"unique;not null;size:255"`
	Department         string `gorm:"size:255"`
	Batch              int    `gorm:"not null"`
	Role               string `gorm:"size:255"`
	IsVerified         bool   `gorm:"not null"`
	IsOnboard          bool   `gorm:"not null"`
	RegistrationStatus string `gorm:"not null;size:100;default:'not_registered'"`
	Email              string `gorm:"unique;not null;size:255"`
	Password           string `gorm:"not null"`
}

type RegisterRequest struct {
	BaseModel   `gorm:"embedded"`
	FullName    string `gorm:"size:255;not null"`
	Department  string `gorm:"size:255;not null"`
	Batch       string `gorm:"size:255;not null"`
	Email       string `gorm:"size:255;unique;not null"`
	PhoneNumber string `gorm:"size:20;not null"`
	Role        string `gorm:"size:255;not null"`
}

type Verification struct {
	BaseModel         `gorm:"embedded"`
	UserID            string `gorm:"not null;size:36"`
	VerificationToken string `gorm:"not null"`
	User              User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	*base = BaseModel{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func MigrateDB(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &RegisterRequest{}, &Verification{})
}

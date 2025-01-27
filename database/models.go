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
	BaseModel          `gorm:"embedded"`
	UserID             string    `gorm:"not null;size:36"`
	VerificationToken  string    `gorm:"size:6"`
	User               User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ResetPasswordToken string    `gorm:"size:36"`
	ExpiresAt          time.Time `gorm:"type:timestamp"`
}

type UserProfile struct {
	UserID       string `gorm:"not null;size:36"`
	ProfileImage string `gorm:"size:255"`
	Headline     string `gorm:"size:100"`
	About        string
	Location     string `gorm:"size:100"`
	Skills       []byte `gorm:"type:jsonb"`
	Interests    []byte `gorm:"type:jsonb"`
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SocialLinks struct {
	ID                 string `gorm:"type:uuid;primaryKey"`
	UserID             string `gorm:"type:uuid;not null"`
	GithubURL          string `gorm:"type:varchar(255)"`
	LinkedinURL        string `gorm:"type:varchar(255)"`
	InstagramURL       string `gorm:"type:varchar(255)"`
	ResumeURL          string `gorm:"type:varchar(255)"`
	PersonalWebsiteURL string `gorm:"type:varchar(255)"`
	User               User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Experience struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	UserID       string `gorm:"type:uuid;not null"`
	CompanyName  string `gorm:"type:varchar(255);not null"`
	Position     string `gorm:"type:varchar(255);not null"`
	StartDate    time.Time
	EndDate      time.Time `gorm:"null"`
	JobType      string    `gorm:"type:varchar(50)"`
	LocationType string    `gorm:"type:varchar(50)"`
	Location     string    `gorm:"type:varchar(255)"`
	User         User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Education struct {
	ID              string    `gorm:"type:uuid;primaryKey"`
	UserID          string    `gorm:"type:uuid;not null"`
	InstitutionName string    `gorm:"type:varchar(255);not null"`
	Course          string    `gorm:"type:varchar(255);not null"`
	Location        string    `gorm:"type:varchar(255)"`
	StartDate       time.Time `gorm:"not null"`
	EndDate         time.Time `gorm:"not null"`
	Grade           string    `gorm:"type:varchar(50)"`
	User            User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
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

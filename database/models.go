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



type RegisterRequest struct {
	BaseModel   `gorm:"embedded"`
	FullName    string `gorm:"size:255;not null"`
	Department  string `gorm:"size:255;not null"`
	Batch       int    `gorm:"not null"`
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

type UserProfile struct {
	BaseModel    `gorm:"embedded"`
	UserID       string `gorm:"size:36;not null"`
	ProfileImage string `gorm:"size:255;null"`
	Headline     string `gorm:"size:100;null"`
	About        string `gorm:"size:500;null"`
	Location     string `gorm:"size:100;null"`
	Skills       []byte `gorm:"type:jsonb;null"`
	Interests    []byte `gorm:"type:jsonb;null"`
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SocialLinks struct {
	BaseModel          `gorm:"embedded"`
	UserID             string `gorm:"size:36;not null"`
	GithubURL          string `gorm:"size:255;null"`
	LinkedinURL        string `gorm:"size:255;null"`
	InstagramURL       string `gorm:"size:255;null"`
	ResumeURL          string `gorm:"size:255;null"`
	PersonalWebsiteURL string `gorm:"size:255;null"`
	User               User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Experience struct {
	BaseModel    `gorm:"embedded"`
	UserID       string     `gorm:"size:36;not null"`
	CompanyName  string     `gorm:"size:255;not null"`
	Position     string     `gorm:"size:255;not null"`
	StartDate    time.Time
	EndDate      *time.Time `gorm:"null"`
	JobType      string     `gorm:"size:50"`
	LocationType string     `gorm:"size:50"`
	Location     string     `gorm:"size:255"`
	User         User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Education struct {
	BaseModel       `gorm:"embedded"`
	UserID          string     `gorm:"size:36;not null"`
	InstitutionName string     `gorm:"size:255;not null"`
	Course          string     `gorm:"size:255;not null"`
	Location        string     `gorm:"size:255"`
	StartDate       time.Time
	EndDate         time.Time `gorm:"null"`
	Grade           string     `gorm:"size:50"`
	User            User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
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
	return db.AutoMigrate(&User{}, &RegisterRequest{}, &Verification{}, &UserProfile{}, &SocialLinks{}, &Experience{}, &Education{})
}
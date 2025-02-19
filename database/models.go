package database

import (
	"errors"
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
	UserName           *string `gorm:"unique;size:255;default:null;null"` // Remove "not null"
	Department         string `gorm:"size:255"`
	Batch              int    `gorm:"not null"`
	Role               string `gorm:"size:255"`
	IsVerified         bool   `gorm:"not null"`
	IsOnboard          bool   `gorm:"not null"`
	RegistrationStatus string `gorm:"not null;size:100;default:'not_registered'"`
	Email              string `gorm:"unique;not null;size:255"`
    Password           string `gorm:"size:255"`        // Remove "not null"
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

type NotificationType string

const (
	NotificationTypeLike    NotificationType = "LIKE"
	NotificationTypeComment NotificationType = "COMMENT"
	NotificationTypeFollow  NotificationType = "FOLLOW"
)

type Post struct {
	BaseModel
	AuthorID string     `gorm:"size:36;not null;index:idx_post_author"`  // Explicit index name
	Content  *string    `gorm:"type:text"`                               // Nullable text content
	Image    *string    `gorm:"size:255"`                                // Nullable image URL
	Author   User       `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Comments []Comment  `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Likes    []Like     `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Comment struct {
	BaseModel
	Content  string `gorm:"type:text;not null"`
	AuthorID string `gorm:"size:36;not null;index:idx_comment_author"`
	PostID   string `gorm:"size:36;not null;index:idx_comment_post"`
	Author   User   `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Post     Post   `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Like struct {
	BaseModel
	PostID string `gorm:"size:36;not null;uniqueIndex:idx_like_post_user"`
	UserID string `gorm:"size:36;not null;uniqueIndex:idx_like_post_user"`
	User   User   `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Post   Post   `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}


type Follow struct {
	BaseModel      // Inherits CreatedAt
	FollowerID  string `gorm:"size:36;primaryKey"`
	FollowingID string `gorm:"size:36;primaryKey"`
	Follower    User   `gorm:"foreignKey:FollowerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Following   User   `gorm:"foreignKey:FollowingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}


type Notification struct {
	BaseModel				   `gorm:"embedded"`
	UserID    string           `gorm:"size:36;not null;index:idx_notification_user"`
	CreatorID string           `gorm:"size:36;not null"`
	Type      NotificationType `gorm:"size:50;not null"`
	Read      bool             `gorm:"not null;default:false"`
	PostID    *string          `gorm:"size:36"`  // Nullable (for FOLLOW-type notifications)
	CommentID *string          `gorm:"size:36"`  // Nullable (for LIKE/FOLLOW notifications)
	
	// Relationships
	User      User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Creator   User     `gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Post      *Post    `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Comment   *Comment `gorm:"foreignKey:CommentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// Composite index for sorting
	CreatedAt time.Time `gorm:"index:idx_notification_user_created,sort:desc"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
    // Generate ID and timestamps via BaseModel's hook
    if err := n.BaseModel.BeforeCreate(tx); err != nil {
        return err
    }
    // Validate notification type
    switch n.Type {
    case NotificationTypeLike, NotificationTypeComment, NotificationTypeFollow:
        return nil
    default:
        return errors.New("invalid notification type")
    }
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
	// Manually create the composite index for sorting
	db.Exec("CREATE INDEX IF NOT EXISTS idx_notification_user_created ON notifications (user_id, created_at DESC)")
	return db.AutoMigrate(&User{}, &RegisterRequest{}, &Verification{}, &UserProfile{}, &SocialLinks{}, &Experience{}, &Education{},
		&Post{},&Comment{},&Like{},&Follow{},&Notification{},
	)
}


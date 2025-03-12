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
	FullName           string  `gorm:"size:255"`
	UserName           *string `gorm:"unique;size:255;default:null;null"`
	Department         string  `gorm:"size:255"`
	Batch              int     `gorm:"not null"`
	Role               string  `gorm:"size:255"`
	IsVerified         bool    `gorm:"not null"`
	IsOnboard          bool    `gorm:"not null"`
	RegistrationStatus string  `gorm:"not null;size:100;default:'not_registered'"`
	Email              string  `gorm:"unique;not null;size:255"`
	Password           string  `gorm:"size:255"`
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
	UserID       string `gorm:"size:36;not null"`
	CompanyName  string `gorm:"size:255;not null"`
	Position     string `gorm:"size:255;not null"`
	StartDate    time.Time
	EndDate      *time.Time `gorm:"null"`
	JobType      string     `gorm:"size:50"`
	LocationType string     `gorm:"size:50"`
	Location     string     `gorm:"size:255"`
	User         User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Education struct {
	BaseModel       `gorm:"embedded"`
	UserID          string `gorm:"size:36;not null"`
	InstitutionName string `gorm:"size:255;not null"`
	Course          string `gorm:"size:255;not null"`
	Location        string `gorm:"size:255"`
	StartDate       time.Time
	EndDate         time.Time `gorm:"null"`
	Grade           string    `gorm:"size:50"`
	User            User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type NotificationType string

const (
	NotificationTypeLike    NotificationType = "LIKE"
	NotificationTypeComment NotificationType = "COMMENT"
	NotificationTypeFollow  NotificationType = "FOLLOW"
)

type Post struct {
	BaseModel
	AuthorID string    `gorm:"size:36;not null;index:idx_post_author"` // Explicit index name
	Content  *string   `gorm:"type:text"`                              // Nullable text content
	Image    *string   `gorm:"size:255"`                               // Nullable image URL
	Author   User      `gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Comments []Comment `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Likes    []Like    `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
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
	BaseModel          // Inherits CreatedAt
	FollowerID  string `gorm:"size:36;primaryKey"`
	FollowingID string `gorm:"size:36;primaryKey"`
	Follower    User   `gorm:"foreignKey:FollowerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Following   User   `gorm:"foreignKey:FollowingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Notification struct {
	BaseModel `gorm:"embedded"`
	UserID    string           `gorm:"size:36;not null;index:idx_notification_user"`
	CreatorID string           `gorm:"size:36;not null"`
	Type      NotificationType `gorm:"size:50;not null"`
	Read      bool             `gorm:"not null;default:false"`
	PostID    *string          `gorm:"size:36"` // Nullable (for FOLLOW-type notifications)
	CommentID *string          `gorm:"size:36"` // Nullable (for LIKE/FOLLOW notifications)

	// Relationships
	User    User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Creator User     `gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Post    *Post    `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Comment *Comment `gorm:"foreignKey:CommentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

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

// Job Portal Models

type Company struct {
	BaseModel   `gorm:"embedded"`
	Name        string `gorm:"size:255;not null"`
	LogoURL     string `gorm:"size:255;null"`
}

type Job struct {
	BaseModel    `gorm:"embedded"`
	Title        string   `gorm:"size:255;not null"`
	PostedBy     string   `gorm:"size:36;not null"`
	CompanyID    string   `gorm:"size:36;not null"`
	Description  string   `gorm:"type:text;not null"`
	Location     string   `gorm:"size:255;not null"`
	Requirements string   `gorm:"type:text;not null"`
	IsOpen       bool     `gorm:"not null;default:true"`
	JobType      string   `gorm:"size:50;not null"`
	ApplyLink    string   `gorm:"size:255;null"`
	
	PostedByUser User     `gorm:"foreignKey:PostedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Company      Company  `gorm:"foreignKey:CompanyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SavedJob struct {
	BaseModel   `gorm:"embedded"`
	UserID      string `gorm:"size:36;not null;uniqueIndex:idx_saved_job_user"`
	JobID       string `gorm:"size:36;not null;uniqueIndex:idx_saved_job_user"`
	
	User        User    `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Job         Job     `gorm:"foreignKey:JobID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type JobReport struct {
	BaseModel     `gorm:"embedded"`
	JobID         string `gorm:"size:36;not null"`
	Reason        string `gorm:"size:255;not null"`
	JobPosterID   string `gorm:"size:36;not null"`
	ReporterID    string `gorm:"size:36;not null"`
	
	Job          Job     `gorm:"foreignKey:JobID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	JobPoster    User    `gorm:"foreignKey:JobPosterID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Reporter     User    `gorm:"foreignKey:ReporterID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	*base = BaseModel{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

type Conversation struct {
	BaseModel
	Participant1ID        string `gorm:"size:36;not null;index"`
	Participant2ID        string `gorm:"size:36;not null;index"`
	LastMessage           string `gorm:"type:text"`
	LastMessageSenderID   string `gorm:"size:36"`
	LastMessageReceiverID string `gorm:"size:36"`
	LastMessageSeen       bool   `gorm:"default:false"`
}

type Message struct {
	BaseModel
	ConversationID string       `gorm:"size:36;not null;index"`
	SenderID       string       `gorm:"type:varchar(36)"`
	ReceiverID     string       `gorm:"type:varchar(36)"`
	Text           string       `gorm:"type:text"`
	Seen           bool         `gorm:"default:false"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type EventType string

const (
    EventTypeCampus EventType = "CAMPUS_EVENT"
    EventTypeAlum   EventType = "ALUM_EVENT"
)

type Event struct {
    BaseModel          `gorm:"embedded"`
    Title              string     `gorm:"size:255;not null"`
    Description        string     `gorm:"type:text;not null"`   // Markdown-ready
    Venue              string     `gorm:"size:255;not null"`
    RegisterLink       string     `gorm:"size:255"`             // Optional
    EventType          EventType  `gorm:"size:20;not null;index:idx_event_type"`
    StartDateTime      time.Time  `gorm:"not null;index:idx_event_time"`
    EndDateTime        time.Time  `gorm:"not null"`
    IsRegistrationOpen bool       `gorm:"not null;default:true"`
    PostedBy           string     `gorm:"size:36;not null;index:idx_event_owner"`
    // Relationships
    User               User       `gorm:"foreignKey:PostedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SavedEvent struct {
    BaseModel   `gorm:"embedded"`
    UserID      string `gorm:"size:36;not null;uniqueIndex:idx_saved_event"`
    EventID     string `gorm:"size:36;not null;uniqueIndex:idx_saved_event"`
    
    User        User  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
    Event       Event `gorm:"foreignKey:EventID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}


func MigrateDB(db *gorm.DB) error {
	// First create tables
	err := db.AutoMigrate(
		&User{}, &RegisterRequest{}, &Verification{}, &UserProfile{}, 
		&SocialLinks{}, &Experience{}, &Education{}, &Post{}, &Comment{}, 
		&Like{}, &Follow{}, &Notification{}, &Conversation{}, &Message{},
		// Add new models
		&Company{}, &Job{}, &SavedJob{}, &JobReport{},&Event{}, &SavedEvent{},
	)
	if err != nil {
		return err
	}

	// Then add constraints
	db.Exec(`ALTER TABLE events ADD CONSTRAINT chk_event_type 
    CHECK (event_type IN ('CAMPUS_EVENT', 'ALUM_EVENT'))`)

	db.Exec(`ALTER TABLE jobs ADD CONSTRAINT chk_job_type 
		CHECK (job_type IN ('Part-Time', 'Full-Time', 'Internship', 'Freelance'))`)

	db.Exec(`ALTER TABLE job_reports ADD CONSTRAINT chk_report_reason 
		CHECK (reason IN ('Fake Job', 'Scam', 'Discriminatory Content', 'Incorrect Information'))`)

	db.Exec("CREATE INDEX IF NOT EXISTS idx_notification_user_created ON notifications (user_id, created_at DESC)")
	
	return nil
}

func CleanupOldNotifications() error {
	now := time.Now().UTC()
	readThreshold := now.AddDate(0, 0, -30)
	unreadThreshold := now.AddDate(0, 0, -90)

	result := Session.Db.Where("read = ? AND created_at < ?", true, readThreshold).Delete(&Notification{})
	if result.Error != nil {
		return result.Error
	}

	result = Session.Db.Where("read = ? AND created_at < ?", false, unreadThreshold).Delete(&Notification{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

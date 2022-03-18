package dbmodels

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Base model that all database tables inherit.
type Model struct {
	ID          *uuid.UUID     `json:"id" gorm:"primaryKey; type:uuid; default:uuid_generate_v4()"`
	CreatedAt   time.Time      `json:"created_at" gorm:"<-:create; autoCreateTime; index; not null"`
	CreatedBy   *User          `json:"-"`
	UpdatedBy   *User          `json:"-"`
	DeletedBy   *User          `json:"-"`
	CreatedByID *uuid.UUID     `json:"created_by_id" gorm:"type:uuid"`
	UpdatedByID *uuid.UUID     `json:"updated_by_id" gorm:"type:uuid"`
	DeletedByID *uuid.UUID     `json:"deleted_by_id" gorm:"type:uuid"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime; not null"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Enable callers to access the base model through an interface.
// This means that setting common metadata like 'created by' or 'updated by' can be abstracted.
func (m *Model) GetModel() *Model {
	return m
}

type Team struct {
	Model
	Slug     *string         `json:"slug" gorm:"<-:create; unique; not null"`
	Name     *string         `json:"name" gorm:"unique; not null"`
	Purpose  *string         `json:"purpose"`
	Metadata []*TeamMetadata `json:"-"`
	Users    []*User         `json:"-" gorm:"many2many:users_teams"`
	Roles    []*Role         `json:"-"`
}

type User struct {
	Model
	Email *string `json:"email" gorm:"unique"`
	Name  *string `json:"name" gorm:"not null" example:"plain english"`
	Teams []*Team `json:"-" gorm:"many2many:users_teams"`
	Roles []*Role `json:"-"`
}

type ApiKey struct {
	Model
	APIKey string    `json:"apikey" gorm:"unique; not null"`
	User   *User     `json:"-"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid; not null"`
}

type TeamMetadata struct {
	Model
	Team   *Team
	TeamID *uuid.UUID `gorm:"uniqueIndex:team_key"`
	Key    string     `gorm:"uniqueIndex:team_key; not null"`
	Value  *string
}

type System struct {
	Model
	Name string `gorm:"uniqueIndex; not null"`
}

type Role struct {
	Model
	System      *System    `gorm:"not null"`
	SystemID    *uuid.UUID `gorm:"not null; index"`
	Resource    string     `gorm:"not null"` // sub-resource at system (maybe not needed if systems are namespaced, e.g. gcp:buckets)
	AccessLevel string     `gorm:"not null"` // read, write, R/W, other combinations per system
	Permission  string     `gorm:"not null"` // allow/deny
	User        *User      // specific user who is allowed or denied access
	Team        *Team      // specific team who is allowed or denied access
	UserID      *uuid.UUID
	TeamID      *uuid.UUID
}

type Synchronization struct {
	Model
}

type AuditLog struct {
	Model
	System            *System          `gorm:"not null"`
	Synchronization   *Synchronization `gorm:"not null"`
	User              *User            // User object, not subject, i.e. which user was affected by the operation.
	Team              *Team
	SystemID          *uuid.UUID
	SynchronizationID *uuid.UUID
	UserID            *uuid.UUID
	TeamID            *uuid.UUID
	Success           bool   `gorm:"not null; index"` // True if operation succeeded
	Action            string `gorm:"not null; index"` // CRUD action
	Message           string `gorm:"not null"`        // User readable success or error message (log line)
}

func (m *AuditLog) Error() string {
	return m.Message
}

func (m *AuditLog) Log() *log.Entry {
	return log.StandardLogger().WithFields(log.Fields{
		"system":         m.System.Name,
		"correlation_id": m.SynchronizationID,
		"user":           m.UserID,
		"team":           m.TeamID,
		"action":         m.Action,
		"success":        m.Success,
	})
}

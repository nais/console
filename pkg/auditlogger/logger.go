package auditlogger

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/nais/console/pkg/dbmodels"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type auditLogger struct {
	db *gorm.DB
}

type AuditLogger interface {
	Logf(action string, corr dbmodels.Correlation, targetSystem dbmodels.System, actor *dbmodels.User, targetTeam *dbmodels.Team, targetUser *dbmodels.User, message string, messageArgs ...interface{}) error
}

func New(db *gorm.DB) AuditLogger {
	return &auditLogger{
		db: db,
	}
}

func (l *auditLogger) Logf(action string, corr dbmodels.Correlation, targetSystem dbmodels.System, actor *dbmodels.User, targetTeam *dbmodels.Team, targetUser *dbmodels.User, message string, messageArgs ...interface{}) error {
	var actorId *uuid.UUID
	var targetTeamId *uuid.UUID
	var targetUserId *uuid.UUID

	if actor != nil && actor.ID != nil {
		actorId = actor.ID
	}

	if targetTeam != nil && targetTeam.ID != nil {
		targetTeamId = targetTeam.ID
	}

	if targetUser != nil && targetUser.ID != nil {
		targetUserId = targetUser.ID
	}

	logEntry := &dbmodels.AuditLog{
		Action:         action,
		Actor:          actor,
		ActorID:        actorId,
		Correlation:    corr,
		CorrelationID:  *corr.ID,
		TargetSystem:   targetSystem,
		TargetSystemID: *targetSystem.ID,
		TargetTeam:     targetTeam,
		TargetUser:     targetUser,
		TargetTeamID:   targetTeamId,
		TargetUserID:   targetUserId,

		Message: fmt.Sprintf(message, messageArgs...),
	}
	err := l.db.Omit(clause.Associations).Create(logEntry).Error
	if err != nil {
		return fmt.Errorf("store audit log line in database: %s", err)
	}

	logEntry.Log().Infof(logEntry.Message)
	return err
}

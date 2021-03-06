package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/graph/generated"
	"github.com/nais/console/pkg/graph/model"
)

func (r *auditLogResolver) TargetSystem(ctx context.Context, obj *dbmodels.AuditLog) (*dbmodels.System, error) {
	var system *dbmodels.System
	err := r.db.Model(&obj).Association("TargetSystem").Find(&system)
	if err != nil {
		return nil, err
	}
	return system, nil
}

func (r *auditLogResolver) Correlation(ctx context.Context, obj *dbmodels.AuditLog) (*dbmodels.Correlation, error) {
	var corr *dbmodels.Correlation
	err := r.db.Model(&obj).Association("Correlation").Find(&corr)
	if err != nil {
		return nil, err
	}
	return corr, nil
}

func (r *auditLogResolver) Actor(ctx context.Context, obj *dbmodels.AuditLog) (*dbmodels.User, error) {
	if obj.ActorID == nil {
		return nil, nil
	}
	var actor *dbmodels.User
	err := r.db.Model(&obj).Association("Actor").Find(&actor)
	if err != nil {
		return nil, err
	}
	return actor, nil
}

func (r *auditLogResolver) TargetUser(ctx context.Context, obj *dbmodels.AuditLog) (*dbmodels.User, error) {
	if obj.TargetUserID == nil {
		return nil, nil
	}
	var targetUser *dbmodels.User
	err := r.db.Model(&obj).Association("TargetUser").Find(&targetUser)
	if err != nil {
		return nil, err
	}
	return targetUser, nil
}

func (r *auditLogResolver) TargetTeam(ctx context.Context, obj *dbmodels.AuditLog) (*dbmodels.Team, error) {
	if obj.TargetTeamID == nil {
		return nil, nil
	}
	var team *dbmodels.Team
	err := r.db.Model(&obj).Association("TargetTeam").Find(&team)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (r *queryResolver) AuditLogs(ctx context.Context, pagination *model.Pagination, query *model.AuditLogsQuery, sort *model.AuditLogsSort) (*model.AuditLogs, error) {
	auditLogs := make([]*dbmodels.AuditLog, 0)

	if sort == nil {
		sort = &model.AuditLogsSort{
			Field:     model.AuditLogSortFieldCreatedAt,
			Direction: model.SortDirectionDesc,
		}
	}
	pageInfo, err := r.paginatedQuery(pagination, query, sort, &dbmodels.AuditLog{}, &auditLogs)
	return &model.AuditLogs{
		PageInfo: pageInfo,
		Nodes:    auditLogs,
	}, err
}

// AuditLog returns generated.AuditLogResolver implementation.
func (r *Resolver) AuditLog() generated.AuditLogResolver { return &auditLogResolver{r} }

type auditLogResolver struct{ *Resolver }

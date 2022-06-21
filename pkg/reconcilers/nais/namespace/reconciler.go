package nais_namespace_reconciler

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/config"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
	"github.com/nais/console/pkg/reconcilers/registry"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

const (
	NaisdCreateNamespace = "create-namespace"
)

type naisdData struct {
	Name       string `json:"name"`
	GcpProject string `json:"gcpProject"` // the user specified "project id"; not the "projects/ID" format
}

type naisdRequest struct {
	Type string    `json:"type"`
	Data naisdData `json:"data"`
}

type namespaceReconciler struct {
	db               *gorm.DB
	config           *jwt.Config
	domain           string
	auditLogger      auditlogger.AuditLogger
	projectParentIDs map[string]string
	credentialsFile  string
	projectID        string
	system           dbmodels.System
}

const (
	Name              = "nais:namespace"
	OpCreateNamespace = "nais:namespace:create-namespace"
)

func init() {
	registry.Register(Name, NewFromConfig)
}

func New(db *gorm.DB, system dbmodels.System, auditLogger auditlogger.AuditLogger, domain, credentialsFile, projectID string, projectParentIDs map[string]string) *namespaceReconciler {
	return &namespaceReconciler{
		db:               db,
		auditLogger:      auditLogger,
		domain:           domain,
		credentialsFile:  credentialsFile,
		projectParentIDs: projectParentIDs,
		projectID:        projectID,
		system:           system,
	}
}

func NewFromConfig(db *gorm.DB, cfg *config.Config, system dbmodels.System, auditLogger auditlogger.AuditLogger) (reconcilers.Reconciler, error) {
	if !cfg.NaisNamespace.Enabled {
		return nil, reconcilers.ErrReconcilerNotEnabled
	}

	return New(db, system, auditLogger, cfg.PartnerDomain, cfg.Google.CredentialsFile, cfg.NaisNamespace.ProjectID, cfg.GCP.ProjectParentIDs), nil
}

func (s *namespaceReconciler) Reconcile(ctx context.Context, input reconcilers.Input) error {
	svc, err := pubsub.NewClient(ctx, s.projectID, option.WithCredentialsFile(s.credentialsFile))
	if err != nil {
		return fmt.Errorf("retrieve pubsub client: %w", err)
	}

	// map of environment -> project ID
	projects := make(map[string]string)

	// read all state variables
	/*
		for _, state := range input.Team.SystemState {
			if state.SystemID != *s.system.ID {
				continue
			}
			if state.Key != dbmodels.SystemStateGoogleProjectID {
				continue
			}
			if state.Environment == nil {
				continue
			}
			projects[*state.Environment] = state.Value
		}

	*/

	for environment := range s.projectParentIDs {
		gcpProjectID := projects[environment]
		if len(gcpProjectID) == 0 {
			return fmt.Errorf("%s: no GCP project created for team '%s' and environment '%s'", OpCreateNamespace, input.Team.Slug, environment)
		}

		err = s.createNamespace(ctx, svc, input.Team, environment, gcpProjectID)
		if err != nil {
			return fmt.Errorf("%s: %w", OpCreateNamespace, err)
		}

		s.auditLogger.Logf(OpCreateNamespace, input.Corr, s.system, nil, &input.Team, nil, "request namespace creation for team '%s' in namespace '%s'", input.Team.Slug, environment)
	}

	return nil
}

func (s *namespaceReconciler) createNamespace(ctx context.Context, pubsubService *pubsub.Client, team dbmodels.Team, environment, gcpProjectID string) error {
	const topicPrefix = "naisd-console-"
	req := &naisdRequest{
		Type: NaisdCreateNamespace,
		Data: naisdData{
			Name:       team.Slug.String(),
			GcpProject: gcpProjectID,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	msg := &pubsub.Message{
		Data: payload,
	}

	topic := topicPrefix + environment
	future := pubsubService.Topic(topic).Publish(ctx, msg)
	<-future.Ready()
	_, err = future.Get(ctx)

	return err
}

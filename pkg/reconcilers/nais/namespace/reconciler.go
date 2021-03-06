package nais_namespace_reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	google_gcp_reconciler "github.com/nais/console/pkg/reconcilers/google/gcp"

	"cloud.google.com/go/pubsub"
	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/config"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
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

type naisNamespaceReconciler struct {
	db               *gorm.DB
	config           *jwt.Config
	domain           string
	auditLogger      auditlogger.AuditLogger
	projectParentIDs map[string]int64
	credentialsFile  string
	projectID        string
	system           dbmodels.System
}

const (
	Name              = "nais:namespace"
	OpCreateNamespace = "nais:namespace:create-namespace"
)

func New(db *gorm.DB, system dbmodels.System, auditLogger auditlogger.AuditLogger, domain, credentialsFile, projectID string, projectParentIDs map[string]int64) *naisNamespaceReconciler {
	return &naisNamespaceReconciler{
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

	return New(db, system, auditLogger, cfg.TenantDomain, cfg.Google.CredentialsFile, cfg.NaisNamespace.ProjectID, cfg.GCP.ProjectParentIDs), nil
}

func (r *naisNamespaceReconciler) Reconcile(ctx context.Context, input reconcilers.Input) error {
	svc, err := pubsub.NewClient(ctx, r.projectID, option.WithCredentialsFile(r.credentialsFile))
	if err != nil {
		return fmt.Errorf("retrieve pubsub client: %w", err)
	}

	gcpSystem := &dbmodels.System{}
	err = r.db.Where("name = ?", google_gcp_reconciler.Name).First(gcpSystem).Error
	if err != nil {
		return fmt.Errorf("unable to load GCP system: %w", err)
	}

	state := &reconcilers.GoogleGcpProjectState{}
	err = dbmodels.LoadSystemState(r.db, *gcpSystem.ID, *input.Team.ID, state)
	if err != nil {
		return fmt.Errorf("unable to load system state for team '%s' in system '%s': %w", input.Team.Slug, r.system.Name, err)
	}

	if len(state.Projects) == 0 {
		return fmt.Errorf("no GCP project state exists for team '%s' yet", input.Team.Slug)
	}

	for environment, project := range state.Projects {
		err = r.createNamespace(ctx, svc, input.Team, environment, project.ProjectID)
		if err != nil {
			return fmt.Errorf("unable to create namespace for project '%s' in environment '%s': %w", project.ProjectID, environment, err)
		}

		// FIXME: Don't create a log entry if the namespace already exists
		r.auditLogger.Logf(OpCreateNamespace, input.Corr, r.system, nil, &input.Team, nil, "request namespace creation for team '%s' in namespace '%s'", input.Team.Slug, environment)
	}

	return nil
}

func (r *naisNamespaceReconciler) System() dbmodels.System {
	return r.system
}

func (r *naisNamespaceReconciler) createNamespace(ctx context.Context, pubsubService *pubsub.Client, team dbmodels.Team, environment, gcpProjectID string) error {
	const topicPrefix = "naisd-console-"
	req := &naisdRequest{
		Type: NaisdCreateNamespace,
		Data: naisdData{
			Name:       string(team.Slug),
			GcpProject: gcpProjectID,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	topic := topicPrefix + environment
	msg := &pubsub.Message{Data: payload}
	future := pubsubService.Topic(topic).Publish(ctx, msg)
	<-future.Ready()
	_, err = future.Get(ctx)
	return err
}

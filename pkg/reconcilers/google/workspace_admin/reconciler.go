package google_workspace_admin_reconciler

import (
	"context"
	"fmt"
	"github.com/nais/console/pkg/google_jwt"
	"net/http"
	"time"

	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/config"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
	"github.com/nais/console/pkg/reconcilers/registry"
	"golang.org/x/oauth2/jwt"
	admin_directory_v1 "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

type gcpReconciler struct {
	logger auditlogger.Logger
	domain string
	config *jwt.Config
}

const (
	Name                    = "google:workspace-admin"
	OpCreate                = "google:workspace-admin:create"
	OpAddMember             = "google:workspace-admin:add-member"
	OpAddMembers            = "google:workspace-admin:add-members"
	OpDeleteMember          = "google:workspace-admin:delete-member"
	OpDeleteMembers         = "google:workspace-admin:delete-members"
	OpAddToGKESecurityGroup = "google:workspace-admin:add-to-gke-security-group"
)

func init() {
	registry.Register(Name, NewFromConfig)
}

func New(logger auditlogger.Logger, domain string, config *jwt.Config) *gcpReconciler {
	return &gcpReconciler{
		logger: logger,
		domain: domain,
		config: config,
	}
}

func NewFromConfig(db *gorm.DB, cfg *config.Config, logger auditlogger.Logger) (reconcilers.Reconciler, error) {
	if !cfg.Google.Enabled {
		return nil, reconcilers.ErrReconcilerNotEnabled
	}

	config, err := google_jwt.GetConfig(cfg.Google.CredentialsFile, cfg.Google.DelegatedUser)

	if err != nil {
		return nil, fmt.Errorf("get google jwt config: %w", err)
	}

	return New(logger, cfg.PartnerDomain, config), nil
}

func (s *gcpReconciler) Reconcile(ctx context.Context, in reconcilers.Input) error {
	client := s.config.Client(ctx)

	srv, err := admin_directory_v1.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("retrieve directory client: %s", err)
	}

	grp, err := s.getOrCreateGroup(srv, in)
	if err != nil {
		return err
	}

	err = s.connectUsers(srv, grp, in)
	if err != nil {
		return s.logger.Errorf(in, OpAddMembers, "add members to group: %s", err)
	}

	err = s.addToGKESecurityGroup(srv, grp, in)
	if err != nil {
		return err
	}

	return nil
}

func (s *gcpReconciler) getOrCreateGroup(srv *admin_directory_v1.Service, in reconcilers.Input) (*admin_directory_v1.Group, error) {
	slug := reconcilers.TeamNamePrefix + *in.Team.Slug
	email := fmt.Sprintf("%s@%s", slug, s.domain)

	grp, err := srv.Groups.Get(email).Do()
	if err == nil {
		return grp, nil
	}

	grp = &admin_directory_v1.Group{
		Id:          slug.String(),
		Email:       email,
		Name:        stringWithFallback(in.Team.Name, fmt.Sprintf("NAIS team '%s'", *in.Team.Slug)),
		Description: stringWithFallback(in.Team.Purpose, fmt.Sprintf("auto-generated by nais console on %s", time.Now().Format(time.RFC1123Z))),
	}

	grp, err = srv.Groups.Insert(grp).Do()
	if err != nil {
		return nil, s.logger.Errorf(in, OpCreate, "create Google Directory group: %s", err)
	}

	s.logger.Logf(in, OpCreate, "successfully created Google Directory group '%s'", grp.Email)

	return grp, nil
}

func (s *gcpReconciler) connectUsers(srv *admin_directory_v1.Service, grp *admin_directory_v1.Group, in reconcilers.Input) error {
	members, err := srv.Members.List(grp.Id).Do()
	if err != nil {
		return s.logger.Errorf(in, OpAddMembers, "list existing members in Google Directory group: %s", err)
	}

	deleteMembers := extraMembers(members.Members, in.Team.Users)
	createUsers := missingUsers(members.Members, in.Team.Users)

	for _, member := range deleteMembers {
		// FIXME: connect audit log with database user, if exists
		err = srv.Members.Delete(grp.Id, member.Id).Do()
		if err != nil {
			return s.logger.UserErrorf(in, OpDeleteMember, nil, "delete member '%s' from Google Directory group '%s': %s", member.Email, grp.Email, err)
		}
		s.logger.UserLogf(in, OpDeleteMember, nil, "deleted member '%s' from Google Directory group '%s'", member.Email, grp.Email)
	}

	if len(deleteMembers) > 0 {
		s.logger.Logf(in, OpDeleteMembers, "all unmanaged members successfully deleted from Google Directory group '%s'", grp.Email)
	}

	for _, user := range createUsers {
		if user.Email == nil {
			continue
		}
		member := &admin_directory_v1.Member{
			Email: *user.Email,
		}
		_, err = srv.Members.Insert(grp.Id, member).Do()
		if err != nil {
			return s.logger.UserErrorf(in, OpAddMember, user, "add member '%s' to Google Directory group '%s': %s", member.Email, grp.Email, err)
		}
		s.logger.UserLogf(in, OpAddMember, user, "added member '%s' to Google Directory group '%s'", member.Email, grp.Email)
	}

	if len(createUsers) > 0 {
		s.logger.Logf(in, OpAddMembers, "all members successfully added to Google Directory group '%s'", grp.Email)
	}

	return nil
}

func (s *gcpReconciler) addToGKESecurityGroup(srv *admin_directory_v1.Service, grp *admin_directory_v1.Group, in reconcilers.Input) error {
	const groupPrefix = "gke-security-groups@"
	groupKey := groupPrefix + s.domain

	member := &admin_directory_v1.Member{
		Email: grp.Email,
	}

	_, err := srv.Members.Insert(groupKey, member).Do()
	if err != nil {
		googleError, ok := err.(*googleapi.Error)
		if ok && googleError.Code == http.StatusConflict {
			return nil
		}
		return s.logger.Errorf(in, OpAddToGKESecurityGroup, "add group '%s' to GKE security group '%s': %s", member.Email, groupKey, err)
	}

	s.logger.Logf(in, OpAddToGKESecurityGroup, "added group '%s' to GKE security group '%s'", member.Email, groupKey)

	return nil
}

// Given a list of Google group members and a list of users,
// return users not present in members directory.
func missingUsers(members []*admin_directory_v1.Member, users []*dbmodels.User) []*dbmodels.User {
	userMap := make(map[string]*dbmodels.User)
	for _, user := range users {
		if user.Email == nil {
			continue
		}
		userMap[*user.Email] = user
	}
	for _, member := range members {
		delete(userMap, member.Email)
	}
	users = make([]*dbmodels.User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}
	return users
}

// Given a list of Google group members and a list of users,
// return members not present in user list.
func extraMembers(members []*admin_directory_v1.Member, users []*dbmodels.User) []*admin_directory_v1.Member {
	memberMap := make(map[string]*admin_directory_v1.Member)
	for _, member := range members {
		memberMap[member.Email] = member
	}
	for _, user := range users {
		if user.Email == nil {
			continue
		}
		delete(memberMap, *user.Email)
	}
	members = make([]*admin_directory_v1.Member, 0, len(memberMap))
	for _, member := range memberMap {
		members = append(members, member)
	}
	return members
}

func stringWithFallback(strp *string, fallback string) string {
	if strp == nil {
		return fallback
	}
	return *strp
}

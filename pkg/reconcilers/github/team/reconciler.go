package github_team_reconciler

import (
	"context"
	"errors"
	"fmt"
	helpers "github.com/nais/console/pkg/console"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v43/github"
	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/config"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
	"github.com/nais/console/pkg/reconcilers/registry"
	"github.com/shurcooL/githubv4"
	"gorm.io/gorm"
)

const (
	Name            = "github:team"
	OpCreate        = "github:team:create"
	OpAddMember     = "github:team:add-member"
	OpAddMembers    = "github:team:add-members"
	OpDeleteMember  = "github:team:delete-member"
	OpDeleteMembers = "github:team:delete-members"
	OpMapSSOUser    = "github:team:map-sso-user"
)

var errGitHubUserNotFound = fmt.Errorf("GitHub user does not exist")

func init() {
	registry.Register(Name, NewFromConfig)
}

func New(logger auditlogger.Logger, org, domain string, teamsService TeamsService, graphClient GraphClient) *gitHubReconciler {
	return &gitHubReconciler{
		logger:       logger,
		org:          org,
		domain:       domain,
		teamsService: teamsService,
		graphClient:  graphClient,
	}
}

func NewFromConfig(db *gorm.DB, cfg *config.Config, logger auditlogger.Logger) (reconcilers.Reconciler, error) {
	if !cfg.GitHub.Enabled {
		return nil, reconcilers.ErrReconcilerNotEnabled
	}

	transport, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		cfg.GitHub.AppID,
		cfg.GitHub.AppInstallationID,
		cfg.GitHub.PrivateKeyPath,
	)
	if err != nil {
		return nil, err
	}

	// Note that both HTTP clients and transports are safe for concurrent use according to the docs,
	// so we can safely reuse them across objects and concurrent synchronizations.
	httpClient := &http.Client{
		Transport: transport,
	}
	restClient := github.NewClient(httpClient)
	graphClient := githubv4.NewClient(httpClient)

	return New(logger, cfg.GitHub.Organization, cfg.PartnerDomain, restClient.Teams, graphClient), nil
}

func (s *gitHubReconciler) Reconcile(ctx context.Context, in reconcilers.Input) error {
	team, err := s.getOrCreateTeam(ctx, in)
	if err != nil {
		return err
	}

	return s.connectUsers(ctx, in, team)
}

func (s *gitHubReconciler) getOrCreateTeam(ctx context.Context, in reconcilers.Input) (*github.Team, error) {
	existingTeam, resp, err := s.teamsService.GetTeamBySlug(ctx, s.org, in.Team.Slug.String())

	if resp == nil && err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return existingTeam, nil
	case http.StatusNotFound:
		break
	default:
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("server raised error: %s: %s", resp.Status, string(body))
	}

	description := helpers.StringWithFallback(in.Team.Purpose, fmt.Sprintf("Team '%v', auto-generated by nais console on %s", in.Team.Name, time.Now().Format(time.RFC1123Z)))

	newTeam := github.NewTeam{
		Name:        in.Team.Slug.String(),
		Description: &description,
	}

	team, resp, err := s.teamsService.CreateTeam(ctx, s.org, newTeam)
	err = httperr(http.StatusCreated, resp, err)
	if err != nil {
		return nil, s.logger.Errorf(in, OpCreate, "create GitHub team '%s': %s", newTeam, err)
	}

	s.logger.Logf(in, OpCreate, "created GitHub team '%s'", newTeam)

	return team, nil
}

func (s *gitHubReconciler) connectUsers(ctx context.Context, in reconcilers.Input, team *github.Team) error {
	userMap, err := s.mapSSOUsers(ctx, in)
	if err != nil {
		return err
	}

	members, err := s.getTeamMembers(ctx, *team.Slug)
	if err != nil {
		return err
	}

	usernames := make([]string, 0, len(userMap))
	for _, username := range userMap {
		usernames = append(usernames, username)
	}
	missing := missingUsers(members, usernames)

	for _, username := range missing {
		// TODO: add user role in membership options?
		// FIXME: connect audit log with database user, if exists
		opts := &github.TeamAddTeamMembershipOptions{}
		_, resp, err := s.teamsService.AddTeamMembershipBySlug(ctx, s.org, *team.Slug, username, opts)
		err = httperr(http.StatusOK, resp, err)
		if err != nil {
			return s.logger.UserErrorf(in, OpAddMember, nil, "add member '%s' to GitHub team '%s': %s", username, *team.Slug, err)
		}
		s.logger.UserLogf(in, OpAddMember, nil, "added member '%s' to GitHub team '%s'", username, *team.Slug)
	}

	s.logger.Logf(in, OpAddMembers, "all members successfully added to GitHub team '%s'", *team.Slug)

	extra := extraMembers(members, usernames)
	for _, username := range extra {
		resp, err := s.teamsService.RemoveTeamMembershipBySlug(ctx, s.org, *team.Slug, username)
		err = httperr(http.StatusNoContent, resp, err)
		if err != nil {
			return s.logger.UserErrorf(in, OpDeleteMember, nil, "remove member '%s' from GitHub team '%s': %s", username, *team.Slug, err)
		}
		s.logger.UserLogf(in, OpDeleteMember, nil, "deleted member '%s' from GitHub team '%s'", username, *team.Slug)
	}

	s.logger.Logf(in, OpDeleteMembers, "all unmanaged members successfully deleted from GitHub team '%s'", *team.Slug)

	return nil
}

func (s *gitHubReconciler) getTeamMembers(ctx context.Context, slug string) ([]*github.User, error) {
	const maxPerPage = 100
	opt := &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: maxPerPage,
		},
	}

	allMembers := make([]*github.User, 0)
	for {
		members, resp, err := s.teamsService.ListTeamMembersBySlug(ctx, s.org, slug, opt)
		err = httperr(http.StatusOK, resp, err)
		if err != nil {
			return nil, err
		}
		allMembers = append(allMembers, members...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allMembers, nil
}

// Given a list of GitHub group members and a list of usernames,
// return usernames not present in members directory.
func missingUsers(members []*github.User, usernames []string) []string {
	userMap := make(map[string]struct{})
	for _, username := range usernames {
		userMap[username] = struct{}{}
	}
	for _, member := range members {
		delete(userMap, member.GetLogin())
	}
	usernames = make([]string, 0, len(userMap))
	for username := range userMap {
		usernames = append(usernames, username)
	}
	return usernames
}

// Given a list of GitHub team members and a list of users,
// return members not present in user list.
func extraMembers(members []*github.User, users []string) []string {
	memberMap := make(map[string]struct{})
	for _, member := range members {
		memberMap[*member.Login] = struct{}{}
	}
	for _, user := range users {
		delete(memberMap, user)
	}

	users = make([]string, 0, len(memberMap))
	for member := range memberMap {
		users = append(users, member)
	}
	return users
}

// Return a mapping of user objects to GitHub usernames.
func (s *gitHubReconciler) mapSSOUsers(ctx context.Context, in reconcilers.Input) (map[*dbmodels.User]string, error) {
	userMap := make(map[*dbmodels.User]string)
	localUsers := helpers.DomainUsers(in.Team.Users, s.domain)

	for _, user := range localUsers {
		githubUsername, err := s.lookupSSOUser(ctx, *user.Email)
		if err == errGitHubUserNotFound {
			s.logger.UserLogf(in, OpMapSSOUser, user, "No GitHub user for email: %s", *user.Email)
			continue
		}
		if err != nil {
			return nil, err
		}
		userMap[user] = githubUsername
	}

	return userMap, nil
}

// Look up a GitHub username from an SSO e-mail address connected to that user account.
func (s *gitHubReconciler) lookupSSOUser(ctx context.Context, email string) (string, error) {
	var query LookupSSOUserQuery

	variables := map[string]interface{}{
		"org":      githubv4.String(s.org),
		"username": githubv4.String(email),
	}

	err := s.graphClient.Query(ctx, &query, variables)
	if err != nil {
		return "", err
	}

	nodes := query.Organization.SamlIdentityProvider.ExternalIdentities.Nodes
	if len(nodes) == 0 {
		return "", errGitHubUserNotFound
	}

	return string(nodes[0].User.Login), nil
}

func httperr(expected int, resp *github.Response, err error) error {
	if err != nil {
		return err
	}
	if resp.StatusCode != expected {
		if resp.Body == nil {
			return errors.New("unknown error")
		}
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("server raised error: %s: %s", resp.Status, string(body))
	}
	return nil
}

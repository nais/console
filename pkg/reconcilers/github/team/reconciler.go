package github_team_reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
	"github.com/shurcooL/githubv4"
)

type GraphClient interface {
	Query(ctx context.Context, q interface{}, variables map[string]interface{}) error
}

type TeamsService interface {
	AddTeamMembershipBySlug(ctx context.Context, org, slug, user string, opts *github.TeamAddTeamMembershipOptions) (*github.Membership, *github.Response, error)
	CreateTeam(ctx context.Context, org string, team github.NewTeam) (*github.Team, *github.Response, error)
	GetTeamBySlug(ctx context.Context, org, slug string) (*github.Team, *github.Response, error)
	ListTeamMembersBySlug(ctx context.Context, org, slug string, opts *github.TeamListTeamMembersOptions) ([]*github.User, *github.Response, error)
	RemoveTeamMembershipBySlug(ctx context.Context, org, slug, user string) (*github.Response, error)
}

// gitHubReconciler creates teams on GitHub and connects users to them.
type gitHubReconciler struct {
	logger       auditlogger.Logger
	teamsService TeamsService
	graphClient  GraphClient
	org          string
}

func New(logger auditlogger.Logger, org string, teamsService TeamsService, graphClient GraphClient) *gitHubReconciler {
	return &gitHubReconciler{
		logger:       logger,
		org:          org,
		teamsService: teamsService,
		graphClient:  graphClient,
	}
}

func (s *gitHubReconciler) Name() string {
	return Name
}

const (
	Name            = "github:team"
	OpCreate        = "github:team:create"
	OpAddMember     = "github:team:add-member"
	OpAddMembers    = "github:team:add-members"
	OpDeleteMember  = "github:team:delete-member"
	OpDeleteMembers = "github:team:delete-members"
)

func (s *gitHubReconciler) Reconcile(ctx context.Context, in reconcilers.Input) error {
	if in.Team == nil || in.Team.Slug == nil {
		return fmt.Errorf("refusing to create team with empty slug")
	}

	team, err := s.getOrCreateTeam(ctx, in)
	if err != nil {
		return err
	}

	err = s.connectUsers(ctx, in, team)

	return err
}

func (s *gitHubReconciler) getOrCreateTeam(ctx context.Context, in reconcilers.Input) (*github.Team, error) {
	existingTeam, _, err := s.teamsService.GetTeamBySlug(ctx, s.org, *in.Team.Slug)

	if err == nil {
		return existingTeam, nil
	}

	description := stringWithFallback(in.Team.Purpose, fmt.Sprintf("Team '%v', auto-generated by nais console on %s", in.Team.Name, time.Now().Format(time.RFC1123Z)))

	newTeam := github.NewTeam{
		Name:        *in.Team.Slug,
		Description: &description,
	}

	team, _, err := s.teamsService.CreateTeam(ctx, s.org, newTeam)
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
		_, _, err = s.teamsService.AddTeamMembershipBySlug(ctx, s.org, *team.Slug, username, opts)
		if err != nil {
			return s.logger.UserErrorf(in, OpAddMember, nil, "add member '%s' to GitHub team '%s': %s", username, *team.Slug, err)
		}
		s.logger.UserLogf(in, OpAddMember, nil, "added member '%s' to GitHub team '%s'", username, *team.Slug)
	}

	s.logger.Logf(in, OpAddMembers, "all members successfully added to GitHub team '%s'", *team.Slug)

	extra := extraMembers(members, usernames)
	for _, username := range extra {
		_, err = s.teamsService.RemoveTeamMembershipBySlug(ctx, s.org, *team.Slug, username)
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

	// connect GitHub usernames with locally defined users.
	userMap := make(map[*dbmodels.User]string)

	for _, user := range in.Team.Users {
		if user.Email == nil {
			continue
		}
		githubUsername, err := s.lookupSSOUser(ctx, *user.Email)
		if err != nil {
			return nil, err
		}
		userMap[user] = githubUsername
	}

	return userMap, nil
}

// Look up a GitHub username from an SSO e-mail address connected to that user account.
func (s *gitHubReconciler) lookupSSOUser(ctx context.Context, email string) (string, error) {
	var query struct {
		Organization struct {
			SamlIdentityProvider struct {
				ExternalIdentities struct {
					Nodes []struct {
						User struct {
							Login githubv4.String
						}
					}
				} `graphql:"externalIdentities(first: 1, userName: $username)"`
			}
		} `graphql:"organization(login: $org)"`
	}

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
		return "", fmt.Errorf("user not found")
	}

	return string(nodes[0].User.Login), nil
}

func stringWithFallback(strp *string, fallback string) string {
	if strp == nil {
		return fallback
	}
	return *strp
}

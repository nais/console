package github_team_reconciler

import (
	"context"
	"github.com/google/go-github/v43/github"
	"github.com/nais/console/pkg/auditlogger"
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
	domain       string
}

type LookupSSOUser struct {
	Login githubv4.String
}

type LookupSSOUserNode struct {
	User LookupSSOUser
}

type LookupSSOUserQuery struct {
	Organization struct {
		SamlIdentityProvider struct {
			ExternalIdentities struct {
				Nodes []LookupSSOUserNode
			} `graphql:"externalIdentities(first: 1, userName: $username)"`
		}
	} `graphql:"organization(login: $org)"`
}

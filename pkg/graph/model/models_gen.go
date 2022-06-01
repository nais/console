// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
	"github.com/nais/console/pkg/dbmodels"
)

// API key type.
type APIKey struct {
	// The API key.
	Apikey string `json:"apikey"`
}

// Input type for API key related operations.
type APIKeyInput struct {
	// ID of a user.
	UserID *uuid.UUID `json:"userId"`
}

// Input for adding users to a team.
type AddUsersToTeamInput struct {
	// List of user IDs that should be added to the team.
	UserIds []*uuid.UUID `json:"userIds"`
	// Team ID that should receive new users.
	TeamID *uuid.UUID `json:"teamId"`
}

// Input for (de)assigning a rule.
type AssignRoleInput struct {
	// The ID of the role.
	RoleID *uuid.UUID `json:"roleId"`
	// The ID of the user.
	UserID *uuid.UUID `json:"userId"`
	// The ID of the team.
	TeamID *uuid.UUID `json:"teamId"`
}

// Audit log collection.
type AuditLogs struct {
	// Object related to pagination of the collection.
	Pagination *Pagination `json:"pagination"`
	// The list of audit log entries in the collection.
	Nodes []*dbmodels.AuditLog `json:"nodes"`
}

// Input for creating a new team.
type CreateTeamInput struct {
	// Team slug.
	Slug *dbmodels.Slug `json:"slug"`
	// Team name.
	Name string `json:"name"`
	// Team purpose.
	Purpose *string `json:"purpose"`
}

// Input for creating a new user.
type CreateUserInput struct {
	// The email address of the new user. Must not already exist, if set.
	Email *string `json:"email"`
	// The name of the new user.
	Name string `json:"name"`
}

// Pagination metadata attached to queries resulting in a collection of data.
type Pagination struct {
	// Total number of results that matches the query.
	Results int `json:"results"`
	// Which record number the returned collection starts at.
	Offset int `json:"offset"`
	// Maximum number of records included in the collection.
	Limit int `json:"limit"`
}

// When querying collections this input is used to control the offset and the page size of the returned slice.
//
// Please note that collections are not stateful, so data added or created in between your paginated requests might not be reflected in the returned result set.
type PaginationInput struct {
	// The offset to start fetching entries.
	Offset int `json:"offset"`
	// Number of entries per page.
	Limit int `json:"limit"`
}

// Input for filtering a collection of audit log entries.
type QueryAuditLogsInput struct {
	// Pagination options.
	Pagination *PaginationInput `json:"pagination"`
	// Filter by team ID.
	TeamID *uuid.UUID `json:"teamId"`
	// Filter by user ID.
	UserID *uuid.UUID `json:"userId"`
	// Filter by system ID.
	SystemID *uuid.UUID `json:"systemId"`
	// Filter by synchronization ID.
	SynchronizationID *uuid.UUID `json:"synchronizationId"`
}

// Input for filtering a collection of roles.
type QueryRolesInput struct {
	// Pagination options.
	Pagination *PaginationInput `json:"pagination"`
	// Filter by role name.
	Name *string `json:"name"`
	// Filter by resource.
	Resource *string `json:"resource"`
	// Filter by access level.
	AccessLevel *string `json:"accessLevel"`
	// Filter by permission.
	Permission *string `json:"permission"`
}

// Input for filtering a collection of teams.
type QueryTeamsInput struct {
	// Pagination options.
	Pagination *PaginationInput `json:"pagination"`
	// Filter by slug.
	Slug *dbmodels.Slug `json:"slug"`
	// Filter by name.
	Name *string `json:"name"`
}

// Input for sorting a collection of teams.
type QueryTeamsSortInput struct {
	// Field to sort by.
	Field TeamSortField `json:"field"`
	// Sort direction.
	Direction SortDirection `json:"direction"`
}

// Input for filtering a collection of users.
type QueryUsersInput struct {
	// Pagination options.
	Pagination *PaginationInput `json:"pagination"`
	// Filter by user email.
	Email *string `json:"email"`
	// Filter by user name.
	Name *string `json:"name"`
}

// Input for sorting a collection of users.
type QueryUsersSortInput struct {
	// Field to sort by.
	Field UserSortField `json:"field"`
	// Sort direction.
	Direction SortDirection `json:"direction"`
}

// Input for removing users from a team.
type RemoveUsersFromTeamInput struct {
	// List of user IDs that should be removed from the team.
	UserIds []*uuid.UUID `json:"userIds"`
	// Team ID that should receive new users.
	TeamID *uuid.UUID `json:"teamId"`
}

// Role collection.
type Roles struct {
	// Object related to pagination of the collection.
	Pagination *Pagination `json:"pagination"`
	// The list of roles in the collection.
	Nodes []*dbmodels.Role `json:"nodes"`
}

type TeamRole struct {
	// ID of the rolebinding
	ID   *uuid.UUID `json:"id"`
	Name string     `json:"name"`
}

// Team collection.
type Teams struct {
	// Object related to pagination of the collection.
	Pagination *Pagination `json:"pagination"`
	// The list of team objects in the collection.
	Nodes []*dbmodels.Team `json:"nodes"`
}

// Input for updating an existing user.
type UpdateUserInput struct {
	// The ID of the existing user.
	ID *uuid.UUID `json:"id"`
	// The updated email address of the user.
	Email *string `json:"email"`
	// The updated name of the user.
	Name *string `json:"name"`
}

// User collection.
type Users struct {
	// Object related to pagination of the collection.
	Pagination *Pagination `json:"pagination"`
	// The list of user objects in the collection.
	Nodes []*dbmodels.User `json:"nodes"`
}

// Direction of the ordering.
type SortDirection string

const (
	// Order ascending.
	SortDirectionAsc SortDirection = "ASC"
	// Order descending.
	SortDirectionDesc SortDirection = "DESC"
)

var AllSortDirection = []SortDirection{
	SortDirectionAsc,
	SortDirectionDesc,
}

func (e SortDirection) IsValid() bool {
	switch e {
	case SortDirectionAsc, SortDirectionDesc:
		return true
	}
	return false
}

func (e SortDirection) String() string {
	return string(e)
}

func (e *SortDirection) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SortDirection(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SortDirection", str)
	}
	return nil
}

func (e SortDirection) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Fields to sort the collection by.
type TeamSortField string

const (
	// Sort by name.
	TeamSortFieldName TeamSortField = "name"
	// Sort by slug.
	TeamSortFieldSlug TeamSortField = "slug"
	// Sort by creation time.
	TeamSortFieldCreatedAt TeamSortField = "createdAt"
)

var AllTeamSortField = []TeamSortField{
	TeamSortFieldName,
	TeamSortFieldSlug,
	TeamSortFieldCreatedAt,
}

func (e TeamSortField) IsValid() bool {
	switch e {
	case TeamSortFieldName, TeamSortFieldSlug, TeamSortFieldCreatedAt:
		return true
	}
	return false
}

func (e TeamSortField) String() string {
	return string(e)
}

func (e *TeamSortField) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TeamSortField(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TeamSortField", str)
	}
	return nil
}

func (e TeamSortField) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Fields to sort the collection by.
type UserSortField string

const (
	// Sort by name.
	UserSortFieldName UserSortField = "name"
	// Sort by email address.
	UserSortFieldEmail UserSortField = "email"
	// Sort by creation time.
	UserSortFieldCreatedAt UserSortField = "createdAt"
)

var AllUserSortField = []UserSortField{
	UserSortFieldName,
	UserSortFieldEmail,
	UserSortFieldCreatedAt,
}

func (e UserSortField) IsValid() bool {
	switch e {
	case UserSortFieldName, UserSortFieldEmail, UserSortFieldCreatedAt:
		return true
	}
	return false
}

func (e UserSortField) String() string {
	return string(e)
}

func (e *UserSortField) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = UserSortField(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid UserSortField", str)
	}
	return nil
}

func (e UserSortField) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

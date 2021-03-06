package azureclient

type GroupResponse struct {
	Value []*Group
}

type Group struct {
	ID              string   `json:"id,omitempty"`
	Description     string   `json:"description,omitempty"`
	DisplayName     string   `json:"displayName,omitempty"`
	GroupTypes      []string `json:"groupTypes,omitempty"`
	MailEnabled     bool     `json:"mailEnabled"`
	MailNickname    string   `json:"mailNickname,omitempty"`
	SecurityEnabled bool     `json:"securityEnabled"`
}

type MemberResponse struct {
	Value []*Member
}

type OwnerResponse struct {
	Value []*Owner
}

type Member struct {
	ID   string `json:"id,omitempty"`
	Mail string `json:"mail,omitempty"`
}

type Owner struct {
	ID                string `json:"id,omitempty"`
	UserPrincipalName string `json:"userPrincipalName,omitempty"`
}

type AddMemberRequest struct {
	ODataID string `json:"@odata.id"`
}

func (m Member) ODataID() string {
	return "https://graph.microsoft.com/v1.0/directoryObjects/" + m.ID
}

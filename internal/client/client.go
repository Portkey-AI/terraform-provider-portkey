package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// JSONNull is a pre-encoded JSON null value for clearing fields via the API.
// Use this with json.RawMessage fields to explicitly send "field": null.
var JSONNull = json.RawMessage("null")

// Client manages communication with the Portkey Admin API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Portkey API client
func NewClient(baseURL, apiKey string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// doRequest performs an HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-portkey-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// WorkspaceDefaults represents the defaults configuration for a workspace
type WorkspaceDefaults struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Workspace represents a Portkey workspace
type Workspace struct {
	ID          string                            `json:"id"`
	Slug        string                            `json:"slug,omitempty"`
	Name        string                            `json:"name"`
	Description string                            `json:"description,omitempty"`
	Defaults    *WorkspaceDefaults                `json:"defaults,omitempty"`
	RateLimits  []IntegrationWorkspaceRateLimits  `json:"rate_limits,omitempty"`
	UsageLimits []IntegrationWorkspaceUsageLimits `json:"usage_limits,omitempty"`
	CreatedAt   time.Time                         `json:"created_at"`
	UpdatedAt   time.Time                         `json:"last_updated_at"`
}

// CreateWorkspaceRequest represents the request to create a workspace
type CreateWorkspaceRequest struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description,omitempty"`
	Defaults    *WorkspaceDefaults                `json:"defaults,omitempty"`
	RateLimits  []IntegrationWorkspaceRateLimits  `json:"rate_limits,omitempty"`
	UsageLimits []IntegrationWorkspaceUsageLimits `json:"usage_limits,omitempty"`
}

// UpdateWorkspaceRequest represents the request to update a workspace.
// UsageLimits and RateLimits use json.RawMessage for three-state semantics:
//   - nil: field omitted from JSON (no change to existing limits)
//   - client.JSONNull: sends "usage_limits": null (clears limits)
//   - marshaled JSON: sends the new limits array
type UpdateWorkspaceRequest struct {
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	Defaults    *WorkspaceDefaults `json:"defaults,omitempty"`
	RateLimits  json.RawMessage    `json:"rate_limits,omitempty"`
	UsageLimits json.RawMessage    `json:"usage_limits,omitempty"`
}

// CreateWorkspace creates a new workspace
func (c *Client) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*Workspace, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/admin/workspaces", req)
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err := json.Unmarshal(respBody, &workspace); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &workspace, nil
}

// GetWorkspace retrieves a workspace by ID
func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/workspaces/"+id, nil)
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err := json.Unmarshal(respBody, &workspace); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &workspace, nil
}

// ListWorkspaces retrieves all workspaces
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/workspaces", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Workspace `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateWorkspace updates a workspace
func (c *Client) UpdateWorkspace(ctx context.Context, id string, req UpdateWorkspaceRequest) (*Workspace, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/admin/workspaces/"+id, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated workspace details since update returns empty response
	workspace, err := c.GetWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("workspace updated but failed to retrieve details: %w", err)
	}

	return workspace, nil
}

// DeleteWorkspaceRequest represents the request to delete a workspace
type DeleteWorkspaceRequest struct {
	Name        string `json:"name"`
	ForceDelete bool   `json:"force_delete,omitempty"`
}

// DeleteWorkspace deletes a workspace
func (c *Client) DeleteWorkspace(ctx context.Context, id string, name string) error {
	req := DeleteWorkspaceRequest{
		Name: name,
	}
	_, err := c.doRequest(ctx, http.MethodDelete, "/admin/workspaces/"+id, req)
	return err
}

// User represents a Portkey user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetUser retrieves a user by ID
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/users/"+id, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &user, nil
}

// ListUsers retrieves all users
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/users", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []User `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Role string `json:"role,omitempty"`
}

// UpdateUser updates a user
func (c *Client) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/admin/users/"+id, req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &user, nil
}

// DeleteUser removes a user
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/admin/users/"+id, nil)
	return err
}

// WorkspaceMember represents a workspace member
type WorkspaceMember struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	Role        string    `json:"role"`
	Email       string    `json:"email,omitempty"`
	FirstName   string    `json:"first_name,omitempty"`
	LastName    string    `json:"last_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// normalizeWorkspaceMember ensures consistent field values
func normalizeWorkspaceMember(member *WorkspaceMember) {
	// If ID is not set, use UserID
	if member.ID == "" && member.UserID != "" {
		member.ID = member.UserID
	}
	// Strip "ws-" prefix from role if present
	member.Role = strings.TrimPrefix(member.Role, "ws-")
}

// workspaceUserItem represents a user in the add users request
type workspaceUserItem struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// addWorkspaceUsersRequest represents the request to add users to a workspace
type addWorkspaceUsersRequest struct {
	Users []workspaceUserItem `json:"users"`
}

// AddWorkspaceMemberRequest represents the request to add a workspace member
type AddWorkspaceMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// AddWorkspaceMember adds a member to a workspace
func (c *Client) AddWorkspaceMember(ctx context.Context, workspaceID string, req AddWorkspaceMemberRequest) (*WorkspaceMember, error) {
	// The API expects { users: [{ id, role }] } format
	addReq := addWorkspaceUsersRequest{
		Users: []workspaceUserItem{
			{
				ID:   req.UserID,
				Role: req.Role,
			},
		},
	}

	path := fmt.Sprintf("/admin/workspaces/%s/users", workspaceID)
	_, err := c.doRequest(ctx, http.MethodPost, path, addReq)
	if err != nil {
		return nil, err
	}

	// The add endpoint doesn't return member details, so we need to fetch them
	member, err := c.GetWorkspaceMember(ctx, workspaceID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user added but failed to retrieve details: %w", err)
	}

	return member, nil
}

// GetWorkspaceMember retrieves a workspace member
func (c *Client) GetWorkspaceMember(ctx context.Context, workspaceID, userID string) (*WorkspaceMember, error) {
	path := fmt.Sprintf("/admin/workspaces/%s/users/%s", workspaceID, userID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var member WorkspaceMember
	if err := json.Unmarshal(respBody, &member); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// API getMember endpoint doesn't return id field, use the queried userID
	if member.ID == "" {
		member.ID = userID
	}
	normalizeWorkspaceMember(&member)
	return &member, nil
}

// ListWorkspaceMembers retrieves all members of a workspace
func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	path := fmt.Sprintf("/admin/workspaces/%s/users", workspaceID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []WorkspaceMember `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Normalize all members
	for i := range response.Data {
		normalizeWorkspaceMember(&response.Data[i])
	}

	return response.Data, nil
}

// UpdateWorkspaceMemberRequest represents the request to update a workspace member
type UpdateWorkspaceMemberRequest struct {
	Role string `json:"role"`
}

// UpdateWorkspaceMember updates a workspace member's role
func (c *Client) UpdateWorkspaceMember(ctx context.Context, workspaceID, userID string, req UpdateWorkspaceMemberRequest) (*WorkspaceMember, error) {
	path := fmt.Sprintf("/admin/workspaces/%s/users/%s", workspaceID, userID)
	_, err := c.doRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated member details
	member, err := c.GetWorkspaceMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("role updated but failed to retrieve details: %w", err)
	}

	return member, nil
}

// RemoveWorkspaceMember removes a member from a workspace
func (c *Client) RemoveWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	path := fmt.Sprintf("/admin/workspaces/%s/users/%s", workspaceID, userID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// UserInvite represents a user invitation
type UserInvite struct {
	ID         string                   `json:"id"`
	Email      string                   `json:"email"`
	Role       string                   `json:"role"`
	Status     string                   `json:"status"`
	Workspaces []WorkspaceInviteDetails `json:"workspaces,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
	ExpiresAt  time.Time                `json:"expires_at"`
}

// WorkspaceInviteDetails represents workspace details in an invitation
type WorkspaceInviteDetails struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// CreateUserInviteRequest represents the request to invite a user
type CreateUserInviteRequest struct {
	Email                  string                   `json:"email"`
	Role                   string                   `json:"role"`
	Workspaces             []WorkspaceInviteDetails `json:"workspaces,omitempty"`
	WorkspaceAPIKeyDetails *APIKeyDetails           `json:"workspace_api_key_details,omitempty"`
}

// APIKeyDetails represents API key configuration for user invites
type APIKeyDetails struct {
	Scopes []string `json:"scopes"`
}

// InviteUser sends an invitation to a user
func (c *Client) InviteUser(ctx context.Context, req CreateUserInviteRequest) (*UserInvite, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/admin/users/invites", req)
	if err != nil {
		return nil, err
	}

	var invite UserInvite
	if err := json.Unmarshal(respBody, &invite); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &invite, nil
}

// GetUserInvite retrieves a user invitation by ID
func (c *Client) GetUserInvite(ctx context.Context, id string) (*UserInvite, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/users/invites/"+id, nil)
	if err != nil {
		return nil, err
	}

	var invite UserInvite
	if err := json.Unmarshal(respBody, &invite); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &invite, nil
}

// ListUserInvites retrieves all user invitations
func (c *Client) ListUserInvites(ctx context.Context) ([]UserInvite, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/admin/users/invites", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []UserInvite `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// DeleteUserInvite deletes a user invitation
func (c *Client) DeleteUserInvite(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/admin/users/invites/"+id, nil)
	return err
}

// Integration represents a Portkey integration (connection to an AI provider)
type Integration struct {
	ID             string                 `json:"id"`
	Slug           string                 `json:"slug"`
	Name           string                 `json:"name"`
	AIProviderID   string                 `json:"ai_provider_id"`
	Description    string                 `json:"description,omitempty"`
	Status         string                 `json:"status"`
	MaskedKey      string                 `json:"masked_key,omitempty"`
	Configurations map[string]interface{} `json:"configurations,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"last_updated_at"`
}

// CreateIntegrationRequest represents the request to create an integration
type CreateIntegrationRequest struct {
	Name           string                 `json:"name"`
	Slug           string                 `json:"slug,omitempty"`
	AIProviderID   string                 `json:"ai_provider_id"`
	Key            string                 `json:"key,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Configurations map[string]interface{} `json:"configurations,omitempty"`
}

// UpdateIntegrationRequest represents the request to update an integration
type UpdateIntegrationRequest struct {
	Name           string                 `json:"name,omitempty"`
	Key            string                 `json:"key,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Configurations map[string]interface{} `json:"configurations,omitempty"`
}

// CreateIntegrationResponse represents the response from creating an integration
type CreateIntegrationResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// CreateIntegration creates a new integration
func (c *Client) CreateIntegration(ctx context.Context, req CreateIntegrationRequest) (*CreateIntegrationResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/integrations", req)
	if err != nil {
		return nil, err
	}

	var response CreateIntegrationResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetIntegration retrieves an integration by slug
func (c *Client) GetIntegration(ctx context.Context, slug string) (*Integration, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/integrations/"+slug, nil)
	if err != nil {
		return nil, err
	}

	var integration Integration
	if err := json.Unmarshal(respBody, &integration); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &integration, nil
}

// ListIntegrations retrieves all integrations
func (c *Client) ListIntegrations(ctx context.Context) ([]Integration, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/integrations", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Integration `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateIntegration updates an integration
func (c *Client) UpdateIntegration(ctx context.Context, slug string, req UpdateIntegrationRequest) (*Integration, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/integrations/"+slug, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated integration details
	return c.GetIntegration(ctx, slug)
}

// DeleteIntegration deletes an integration
func (c *Client) DeleteIntegration(ctx context.Context, slug string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/integrations/"+slug, nil)
	return err
}

// APIKeyDefaults represents the defaults configuration for an API key
type APIKeyDefaults struct {
	Metadata            map[string]string `json:"metadata,omitempty"`
	ConfigID            string            `json:"config_id,omitempty"`
	AllowConfigOverride *bool             `json:"allow_config_override,omitempty"`
}

// APIKey represents a Portkey API key
type APIKey struct {
	ID             string          `json:"id"`
	Key            string          `json:"key,omitempty"` // Only returned on creation
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	Type           string          `json:"type"` // organisation-service, workspace-service, workspace-user
	OrganisationID string          `json:"organisation_id"`
	WorkspaceID    string          `json:"workspace_id,omitempty"`
	UserID         string          `json:"user_id,omitempty"`
	Status         string          `json:"status"`
	CreationMode   string          `json:"creation_mode,omitempty"`
	RateLimits     []RateLimit     `json:"rate_limits,omitempty"`
	UsageLimits    *UsageLimits    `json:"usage_limits,omitempty"`
	Scopes         []string        `json:"scopes,omitempty"`
	Defaults       *APIKeyDefaults `json:"defaults,omitempty"`
	AlertEmails    []string        `json:"alert_emails,omitempty"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"last_updated_at"`
}

// RateLimit represents a rate limit configuration
type RateLimit struct {
	Type  string `json:"type"` // requests
	Unit  string `json:"unit"` // rpm, rpd
	Value int    `json:"value"`
}

// UsageLimits represents usage limit configuration for API keys.
// Uses the same field names as workspace usage limits (credit_limit, periodic_reset, alert_threshold).
type UsageLimits struct {
	CreditLimit    *int   `json:"credit_limit,omitempty"`
	AlertThreshold *int   `json:"alert_threshold,omitempty"`
	PeriodicReset  string `json:"periodic_reset,omitempty"` // monthly, weekly
}

// CreateAPIKeyRequest represents the request to create an API key
type CreateAPIKeyRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	UserID      string          `json:"user_id,omitempty"` // Required for user sub-type
	RateLimits  []RateLimit     `json:"rate_limits,omitempty"`
	UsageLimits *UsageLimits    `json:"usage_limits,omitempty"`
	Scopes      []string        `json:"scopes,omitempty"`
	Defaults    *APIKeyDefaults `json:"defaults,omitempty"`
	AlertEmails []string        `json:"alert_emails,omitempty"`
	ExpiresAt   string          `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents the response from creating an API key
type CreateAPIKeyResponse struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Object string `json:"object"`
}

// UpdateAPIKeyRequest represents the request to update an API key.
// UsageLimits and RateLimits use json.RawMessage for three-state semantics:
//   - nil: field omitted from JSON (no change to existing limits)
//   - client.JSONNull: sends "usage_limits": null (clears limits)
//   - marshaled JSON: sends the new limits object/array
type UpdateAPIKeyRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	RateLimits  json.RawMessage `json:"rate_limits,omitempty"`
	UsageLimits json.RawMessage `json:"usage_limits,omitempty"`
	Scopes      []string        `json:"scopes,omitempty"`
	Defaults    *APIKeyDefaults `json:"defaults,omitempty"`
	AlertEmails []string        `json:"alert_emails,omitempty"`
}

// CreateAPIKey creates a new API key
// keyType: "organisation" or "workspace"
// subType: "service" or "user"
func (c *Client) CreateAPIKey(ctx context.Context, keyType, subType string, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	path := fmt.Sprintf("/api-keys/%s/%s", keyType, subType)
	respBody, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var response CreateAPIKeyResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetAPIKey retrieves an API key by ID
func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/api-keys/"+id, nil)
	if err != nil {
		return nil, err
	}

	var apiKey APIKey
	if err := json.Unmarshal(respBody, &apiKey); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &apiKey, nil
}

// ListAPIKeys retrieves all API keys
func (c *Client) ListAPIKeys(ctx context.Context, workspaceID string) ([]APIKey, error) {
	path := "/api-keys"
	if workspaceID != "" {
		path = fmt.Sprintf("/api-keys?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []APIKey `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateAPIKey updates an API key
func (c *Client) UpdateAPIKey(ctx context.Context, id string, req UpdateAPIKeyRequest) (*APIKey, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/api-keys/"+id, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated API key details
	return c.GetAPIKey(ctx, id)
}

// DeleteAPIKey deletes an API key
func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api-keys/"+id, nil)
	return err
}

// Provider represents a Portkey provider (virtual key)
type Provider struct {
	ID            string                 `json:"id"`
	Slug          string                 `json:"slug"`
	Name          string                 `json:"name"`
	AIProviderID  string                 `json:"ai_provider_name,omitempty"`
	IntegrationID string                 `json:"integration_id,omitempty"`
	WorkspaceID   string                 `json:"workspace_id,omitempty"`
	Status        string                 `json:"status"`
	Note          string                 `json:"note,omitempty"`
	ModelConfig   map[string]interface{} `json:"model_config,omitempty"`
	RateLimits    []RateLimit            `json:"rate_limits,omitempty"`
	UsageLimits   *UsageLimits           `json:"usage_limits,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
}

// CreateProviderRequest represents the request to create a provider
type CreateProviderRequest struct {
	Name          string                 `json:"name"`
	Slug          string                 `json:"slug,omitempty"`
	WorkspaceID   string                 `json:"workspace_id"`
	IntegrationID string                 `json:"integration_id"`
	Note          string                 `json:"note,omitempty"`
	ModelConfig   map[string]interface{} `json:"model_config,omitempty"`
	RateLimits    []RateLimit            `json:"rate_limits,omitempty"`
	UsageLimits   *UsageLimits           `json:"usage_limits,omitempty"`
}

// CreateProviderResponse represents the response from creating a provider
type CreateProviderResponse struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Object string `json:"object"`
}

// UpdateProviderRequest represents the request to update a provider
type UpdateProviderRequest struct {
	Name        string                 `json:"name,omitempty"`
	WorkspaceID string                 `json:"workspace_id"`
	Note        string                 `json:"note,omitempty"`
	ModelConfig map[string]interface{} `json:"model_config,omitempty"`
	RateLimits  []RateLimit            `json:"rate_limits,omitempty"`
	UsageLimits *UsageLimits           `json:"usage_limits,omitempty"`
}

// CreateProvider creates a new provider
func (c *Client) CreateProvider(ctx context.Context, req CreateProviderRequest) (*CreateProviderResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/providers", req)
	if err != nil {
		return nil, err
	}

	var response CreateProviderResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetProvider retrieves a provider by ID
func (c *Client) GetProvider(ctx context.Context, id, workspaceID string) (*Provider, error) {
	path := fmt.Sprintf("/providers/%s?workspace_id=%s", id, workspaceID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var provider Provider
	if err := json.Unmarshal(respBody, &provider); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &provider, nil
}

// ListProviders retrieves all providers for a workspace
func (c *Client) ListProviders(ctx context.Context, workspaceID string) ([]Provider, error) {
	path := "/providers"
	if workspaceID != "" {
		path = fmt.Sprintf("/providers?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Provider `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateProvider updates a provider
func (c *Client) UpdateProvider(ctx context.Context, id string, req UpdateProviderRequest) (*Provider, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/providers/"+id, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated provider details
	return c.GetProvider(ctx, id, req.WorkspaceID)
}

// DeleteProvider deletes a provider
func (c *Client) DeleteProvider(ctx context.Context, id, workspaceID string) error {
	path := fmt.Sprintf("/providers/%s?workspace_id=%s", id, workspaceID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// Config represents a Portkey config
type Config struct {
	ID             string                 `json:"id"`
	Slug           string                 `json:"slug"`
	Name           string                 `json:"name"`
	Config         map[string]interface{} `json:"-"` // Parsed config map
	ConfigRaw      string                 `json:"-"` // Raw config string
	WorkspaceID    string                 `json:"workspace_id"`
	OrganisationID string                 `json:"organisation_id"`
	IsDefault      int                    `json:"is_default"`
	Status         string                 `json:"status"`
	OwnerID        string                 `json:"owner_id,omitempty"`
	UpdatedBy      string                 `json:"updated_by,omitempty"`
	Format         string                 `json:"format,omitempty"`
	Type           string                 `json:"type,omitempty"`
	VersionID      string                 `json:"version_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"last_updated_at"`
}

// CreateConfigRequest represents the request to create a config
type CreateConfigRequest struct {
	Name        string                 `json:"name"`
	Config      map[string]interface{} `json:"config"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	IsDefault   *int                   `json:"isDefault,omitempty"`
}

// CreateConfigResponse represents the response from creating a config
type CreateConfigResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	VersionID string `json:"version_id"`
}

// UpdateConfigRequest represents the request to update a config
type UpdateConfigRequest struct {
	Name   string                 `json:"name,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
	Status string                 `json:"status,omitempty"`
}

// UpdateConfigResponse represents the response from updating a config
type UpdateConfigResponse struct {
	VersionID string `json:"version_id"`
}

// CreateConfig creates a new config
func (c *Client) CreateConfig(ctx context.Context, req CreateConfigRequest) (*CreateConfigResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/configs", req)
	if err != nil {
		return nil, err
	}

	var response CreateConfigResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// configAPIResponse is used for unmarshaling API responses with flexible config field
type configAPIResponse struct {
	ID             string      `json:"id"`
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	Config         interface{} `json:"config"` // Can be string or object
	WorkspaceID    string      `json:"workspace_id"`
	OrganisationID string      `json:"organisation_id"`
	IsDefault      int         `json:"is_default"`
	Status         string      `json:"status"`
	OwnerID        string      `json:"owner_id,omitempty"`
	UpdatedBy      string      `json:"updated_by,omitempty"`
	Format         string      `json:"format,omitempty"`
	Type           string      `json:"type,omitempty"`
	VersionID      string      `json:"version_id,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"last_updated_at"`
}

// GetConfig retrieves a config by slug
func (c *Client) GetConfig(ctx context.Context, slug string) (*Config, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/configs/"+slug, nil)
	if err != nil {
		return nil, err
	}

	var apiResp configAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	config := &Config{
		ID:             apiResp.ID,
		Slug:           apiResp.Slug,
		Name:           apiResp.Name,
		WorkspaceID:    apiResp.WorkspaceID,
		OrganisationID: apiResp.OrganisationID,
		IsDefault:      apiResp.IsDefault,
		Status:         apiResp.Status,
		OwnerID:        apiResp.OwnerID,
		UpdatedBy:      apiResp.UpdatedBy,
		Format:         apiResp.Format,
		Type:           apiResp.Type,
		VersionID:      apiResp.VersionID,
		CreatedAt:      apiResp.CreatedAt,
		UpdatedAt:      apiResp.UpdatedAt,
	}

	// Handle config field which can be a string (JSON) or object
	switch v := apiResp.Config.(type) {
	case string:
		config.ConfigRaw = v
		// Parse string to map
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(v), &configMap); err == nil {
			config.Config = configMap
		}
	case map[string]interface{}:
		config.Config = v
		// Convert to string
		if configBytes, err := json.Marshal(v); err == nil {
			config.ConfigRaw = string(configBytes)
		}
	}

	return config, nil
}

// ListConfigs retrieves all configs
func (c *Client) ListConfigs(ctx context.Context, workspaceID string) ([]Config, error) {
	path := "/configs"
	if workspaceID != "" {
		path = fmt.Sprintf("/configs?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Config `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateConfig updates a config
func (c *Client) UpdateConfig(ctx context.Context, slug string, req UpdateConfigRequest) (*UpdateConfigResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/configs/"+slug, req)
	if err != nil {
		return nil, err
	}

	var response UpdateConfigResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// DeleteConfig deletes a config
func (c *Client) DeleteConfig(ctx context.Context, slug string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/configs/"+slug, nil)
	return err
}

// Prompt represents a Portkey prompt
type Prompt struct {
	ID                       string                 `json:"id"`
	Slug                     string                 `json:"slug"`
	Name                     string                 `json:"name"`
	CollectionID             string                 `json:"collection_id"`
	String                   string                 `json:"string"`
	Parameters               map[string]interface{} `json:"parameters,omitempty"`
	Model                    string                 `json:"model,omitempty"`
	VirtualKey               string                 `json:"virtual_key,omitempty"`
	Functions                []interface{}          `json:"functions,omitempty"`
	Tools                    []interface{}          `json:"tools,omitempty"`
	ToolChoice               interface{}            `json:"tool_choice,omitempty"`
	TemplateMetadata         map[string]interface{} `json:"template_metadata,omitempty"`
	IsRawTemplate            int                    `json:"is_raw_template"`
	PromptVersion            int                    `json:"prompt_version"`
	PromptVersionID          string                 `json:"prompt_version_id,omitempty"`
	PromptVersionStatus      string                 `json:"prompt_version_status,omitempty"`
	PromptVersionDescription string                 `json:"prompt_version_description,omitempty"`
	Status                   string                 `json:"status"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"last_updated_at"`
}

// CreatePromptRequest represents the request to create a prompt
type CreatePromptRequest struct {
	Name               string                 `json:"name"`
	CollectionID       string                 `json:"collection_id"`
	String             string                 `json:"string"`
	Parameters         map[string]interface{} `json:"parameters"`
	Model              string                 `json:"model,omitempty"`
	VirtualKey         string                 `json:"virtual_key"`
	VersionDescription string                 `json:"version_description,omitempty"`
	TemplateMetadata   map[string]interface{} `json:"template_metadata,omitempty"`
}

// CreatePromptResponse represents the response from creating a prompt
type CreatePromptResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	VersionID string `json:"version_id"`
}

// UpdatePromptRequest represents the request to update a prompt.
// Note: parameters, model, virtual_key, and is_raw_template use non-omitempty
// tags or pointer types to ensure they're included in version-creating updates.
// The API requires these fields when creating a new version.
type UpdatePromptRequest struct {
	Name               string                 `json:"name,omitempty"`
	CollectionID       string                 `json:"collection_id,omitempty"`
	String             string                 `json:"string,omitempty"`
	Parameters         map[string]interface{} `json:"parameters"`
	Model              string                 `json:"model,omitempty"`
	VirtualKey         string                 `json:"virtual_key,omitempty"`
	VersionDescription string                 `json:"version_description,omitempty"`
	TemplateMetadata   map[string]interface{} `json:"template_metadata,omitempty"`
	IsRawTemplate      *int                   `json:"is_raw_template,omitempty"`
}

// UpdatePromptResponse represents the response from updating a prompt
type UpdatePromptResponse struct {
	ID              string `json:"id,omitempty"`
	Slug            string `json:"slug,omitempty"`
	PromptVersionID string `json:"prompt_version_id,omitempty"`
}

// CreatePrompt creates a new prompt
func (c *Client) CreatePrompt(ctx context.Context, req CreatePromptRequest) (*CreatePromptResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/prompts", req)
	if err != nil {
		return nil, err
	}

	var response CreatePromptResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetPrompt retrieves a prompt by slug or ID
func (c *Client) GetPrompt(ctx context.Context, slugOrID string, version string) (*Prompt, error) {
	path := "/prompts/" + slugOrID
	if version != "" {
		path += "?version=" + version
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var prompt Prompt
	if err := json.Unmarshal(respBody, &prompt); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &prompt, nil
}

// ListPrompts retrieves all prompts
func (c *Client) ListPrompts(ctx context.Context, workspaceID, collectionID string) ([]Prompt, error) {
	path := "/prompts"
	params := []string{}
	if workspaceID != "" {
		params = append(params, "workspace_id="+workspaceID)
	}
	if collectionID != "" {
		params = append(params, "collection_id="+collectionID)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Prompt `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdatePrompt updates a prompt
func (c *Client) UpdatePrompt(ctx context.Context, slugOrID string, req UpdatePromptRequest) (*UpdatePromptResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/prompts/"+slugOrID, req)
	if err != nil {
		return nil, err
	}

	var response UpdatePromptResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		// Name-only updates return empty JSON {}
		return &UpdatePromptResponse{}, nil
	}

	return &response, nil
}

// MakePromptVersionDefault makes a specific version the default
func (c *Client) MakePromptVersionDefault(ctx context.Context, slugOrID string, version int) error {
	req := map[string]int{"version": version}
	_, err := c.doRequest(ctx, http.MethodPut, "/prompts/"+slugOrID+"/makeDefault", req)
	return err
}

// DeletePrompt deletes a prompt
func (c *Client) DeletePrompt(ctx context.Context, slugOrID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/prompts/"+slugOrID, nil)
	return err
}

// PromptPartial represents a Portkey prompt partial
type PromptPartial struct {
	ID                         string    `json:"id"`
	Slug                       string    `json:"slug"`
	Name                       string    `json:"name"`
	String                     string    `json:"string"`
	Status                     string    `json:"status"`
	Version                    int       `json:"version"`
	PromptPartialVersionID     string    `json:"prompt_partial_version_id,omitempty"`
	PromptPartialVersionStatus string    `json:"prompt_partial_version_status,omitempty"`
	VersionDescription         string    `json:"version_description,omitempty"`
	CollectionID               string    `json:"collection_id,omitempty"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"last_updated_at"`
}

// CreatePromptPartialRequest represents the request to create a prompt partial
type CreatePromptPartialRequest struct {
	Name               string `json:"name"`
	String             string `json:"string"`
	WorkspaceID        string `json:"workspace_id,omitempty"`
	VersionDescription string `json:"version_description,omitempty"`
}

// CreatePromptPartialResponse represents the response from creating a prompt partial
type CreatePromptPartialResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	VersionID string `json:"version_id"`
}

// UpdatePromptPartialRequest represents the request to update a prompt partial
type UpdatePromptPartialRequest struct {
	Name               string `json:"name,omitempty"`
	String             string `json:"string,omitempty"`
	VersionDescription string `json:"version_description,omitempty"`
}

// UpdatePromptPartialResponse represents the response from updating a prompt partial
type UpdatePromptPartialResponse struct {
	ID                     string `json:"id,omitempty"`
	Slug                   string `json:"slug,omitempty"`
	PromptPartialVersionID string `json:"prompt_partial_version_id,omitempty"`
}

// CreatePromptPartial creates a new prompt partial
func (c *Client) CreatePromptPartial(ctx context.Context, req CreatePromptPartialRequest) (*CreatePromptPartialResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/prompts/partials", req)
	if err != nil {
		return nil, err
	}

	var response CreatePromptPartialResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetPromptPartial retrieves a prompt partial by slug or ID
func (c *Client) GetPromptPartial(ctx context.Context, slugOrID string, version string) (*PromptPartial, error) {
	path := "/prompts/partials/" + slugOrID
	if version != "" {
		path += "?version=" + version
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var partial PromptPartial
	if err := json.Unmarshal(respBody, &partial); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &partial, nil
}

// ListPromptPartials retrieves all prompt partials
func (c *Client) ListPromptPartials(ctx context.Context, workspaceID string) ([]PromptPartial, error) {
	path := "/prompts/partials"
	if workspaceID != "" {
		path += "?workspace_id=" + workspaceID
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []PromptPartial `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdatePromptPartial updates a prompt partial
func (c *Client) UpdatePromptPartial(ctx context.Context, slugOrID string, req UpdatePromptPartialRequest) (*UpdatePromptPartialResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPut, "/prompts/partials/"+slugOrID, req)
	if err != nil {
		return nil, err
	}

	var response UpdatePromptPartialResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		// Name-only updates may return empty JSON {}
		return &UpdatePromptPartialResponse{}, nil
	}

	return &response, nil
}

// MakePromptPartialVersionDefault makes a specific version the default
func (c *Client) MakePromptPartialVersionDefault(ctx context.Context, slugOrID string, version int) error {
	req := map[string]int{"version": version}
	_, err := c.doRequest(ctx, http.MethodPut, "/prompts/partials/"+slugOrID+"/makeDefault", req)
	return err
}

// DeletePromptPartial deletes a prompt partial
func (c *Client) DeletePromptPartial(ctx context.Context, slugOrID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/prompts/partials/"+slugOrID, nil)
	return err
}

// Guardrail represents a Portkey guardrail
type Guardrail struct {
	ID             string                 `json:"id"`
	Slug           string                 `json:"slug"`
	Name           string                 `json:"name"`
	OrganisationID string                 `json:"organisation_id,omitempty"`
	WorkspaceID    string                 `json:"workspace_id,omitempty"`
	Checks         []GuardrailCheck       `json:"checks"`
	Actions        map[string]interface{} `json:"actions"`
	Status         string                 `json:"status"`
	VersionID      string                 `json:"version_id,omitempty"`
	OwnerID        string                 `json:"owner_id,omitempty"`
	UpdatedBy      string                 `json:"updated_by,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"last_updated_at"`
}

// GuardrailCheck represents a check in a guardrail
type GuardrailCheck struct {
	ID         string                 `json:"id"`
	IsEnabled  *bool                  `json:"is_enabled,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// CreateGuardrailRequest represents the request to create a guardrail
type CreateGuardrailRequest struct {
	Name           string                 `json:"name"`
	WorkspaceID    string                 `json:"workspace_id,omitempty"`
	OrganisationID string                 `json:"organisation_id,omitempty"`
	Checks         []GuardrailCheck       `json:"checks"`
	Actions        map[string]interface{} `json:"actions"`
}

// CreateGuardrailResponse represents the response from creating a guardrail
type CreateGuardrailResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	VersionID string `json:"version_id"`
}

// UpdateGuardrailRequest represents the request to update a guardrail
type UpdateGuardrailRequest struct {
	Name    string                 `json:"name,omitempty"`
	Checks  []GuardrailCheck       `json:"checks,omitempty"`
	Actions map[string]interface{} `json:"actions,omitempty"`
}

// CreateGuardrail creates a new guardrail
func (c *Client) CreateGuardrail(ctx context.Context, req CreateGuardrailRequest) (*CreateGuardrailResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/guardrails", req)
	if err != nil {
		return nil, err
	}

	var response CreateGuardrailResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetGuardrail retrieves a guardrail by slug or ID
func (c *Client) GetGuardrail(ctx context.Context, slugOrID string) (*Guardrail, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/guardrails/"+slugOrID, nil)
	if err != nil {
		return nil, err
	}

	var guardrail Guardrail
	if err := json.Unmarshal(respBody, &guardrail); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &guardrail, nil
}

// ListGuardrails retrieves all guardrails
func (c *Client) ListGuardrails(ctx context.Context, workspaceID string) ([]Guardrail, error) {
	path := "/guardrails"
	if workspaceID != "" {
		path = fmt.Sprintf("/guardrails?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Guardrail `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateGuardrail updates a guardrail
func (c *Client) UpdateGuardrail(ctx context.Context, slugOrID string, req UpdateGuardrailRequest) (*Guardrail, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/guardrails/"+slugOrID, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated guardrail details
	return c.GetGuardrail(ctx, slugOrID)
}

// DeleteGuardrail deletes a guardrail
func (c *Client) DeleteGuardrail(ctx context.Context, slugOrID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/guardrails/"+slugOrID, nil)
	return err
}

// PolicyCondition represents a condition in a policy
type PolicyCondition struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PolicyGroupBy represents a group by field in a policy
type PolicyGroupBy struct {
	Key string `json:"key"`
}

// UsageLimitsPolicy represents a Portkey usage limits policy
type UsageLimitsPolicy struct {
	ID             string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	Conditions     []PolicyCondition `json:"conditions"`
	GroupBy        []PolicyGroupBy   `json:"group_by"`
	Type           string            `json:"type"`
	CreditLimit    float64           `json:"credit_limit"`
	AlertThreshold *float64          `json:"alert_threshold,omitempty"`
	PeriodicReset  string            `json:"periodic_reset,omitempty"`
	Status         string            `json:"status"`
	WorkspaceID    string            `json:"workspace_id"`
	OrganisationID string            `json:"organisation_id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"last_updated_at"`
}

// CreateUsageLimitsPolicyRequest represents the request to create a usage limits policy
type CreateUsageLimitsPolicyRequest struct {
	Name           string            `json:"name,omitempty"`
	WorkspaceID    string            `json:"workspace_id,omitempty"`
	OrganisationID string            `json:"organisation_id,omitempty"`
	Conditions     []PolicyCondition `json:"conditions"`
	GroupBy        []PolicyGroupBy   `json:"group_by"`
	Type           string            `json:"type"`
	CreditLimit    float64           `json:"credit_limit"`
	AlertThreshold *float64          `json:"alert_threshold,omitempty"`
	PeriodicReset  string            `json:"periodic_reset,omitempty"`
}

// UpdateUsageLimitsPolicyRequest represents the request to update a usage limits policy
type UpdateUsageLimitsPolicyRequest struct {
	Name           string   `json:"name,omitempty"`
	CreditLimit    *float64 `json:"credit_limit,omitempty"`
	AlertThreshold *float64 `json:"alert_threshold,omitempty"`
	Status         string   `json:"status,omitempty"`
}

// CreateUsageLimitsPolicyResponse represents the response from creating a usage limits policy
type CreateUsageLimitsPolicyResponse struct {
	ID string `json:"id"`
}

// CreateUsageLimitsPolicy creates a new usage limits policy
func (c *Client) CreateUsageLimitsPolicy(ctx context.Context, req CreateUsageLimitsPolicyRequest) (*CreateUsageLimitsPolicyResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/policies/usage-limits", req)
	if err != nil {
		return nil, err
	}

	var response CreateUsageLimitsPolicyResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetUsageLimitsPolicy retrieves a usage limits policy by ID
func (c *Client) GetUsageLimitsPolicy(ctx context.Context, id string) (*UsageLimitsPolicy, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/policies/usage-limits/"+id, nil)
	if err != nil {
		return nil, err
	}

	var policy UsageLimitsPolicy
	if err := json.Unmarshal(respBody, &policy); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &policy, nil
}

// ListUsageLimitsPolicies retrieves all usage limits policies
func (c *Client) ListUsageLimitsPolicies(ctx context.Context, workspaceID string) ([]UsageLimitsPolicy, error) {
	path := "/policies/usage-limits"
	if workspaceID != "" {
		path = fmt.Sprintf("/policies/usage-limits?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []UsageLimitsPolicy `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateUsageLimitsPolicy updates a usage limits policy
func (c *Client) UpdateUsageLimitsPolicy(ctx context.Context, id string, req UpdateUsageLimitsPolicyRequest) (*UsageLimitsPolicy, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/policies/usage-limits/"+id, req)
	if err != nil {
		return nil, err
	}

	return c.GetUsageLimitsPolicy(ctx, id)
}

// DeleteUsageLimitsPolicy deletes a usage limits policy
func (c *Client) DeleteUsageLimitsPolicy(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/policies/usage-limits/"+id, nil)
	return err
}

// RateLimitsPolicy represents a Portkey rate limits policy
type RateLimitsPolicy struct {
	ID             string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	Conditions     []PolicyCondition `json:"conditions"`
	GroupBy        []PolicyGroupBy   `json:"group_by"`
	Type           string            `json:"type"`
	Unit           string            `json:"unit"`
	Value          float64           `json:"value"`
	Status         string            `json:"status"`
	WorkspaceID    string            `json:"workspace_id"`
	OrganisationID string            `json:"organisation_id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"last_updated_at"`
}

// CreateRateLimitsPolicyRequest represents the request to create a rate limits policy
type CreateRateLimitsPolicyRequest struct {
	Name           string            `json:"name,omitempty"`
	WorkspaceID    string            `json:"workspace_id,omitempty"`
	OrganisationID string            `json:"organisation_id,omitempty"`
	Conditions     []PolicyCondition `json:"conditions"`
	GroupBy        []PolicyGroupBy   `json:"group_by"`
	Type           string            `json:"type"`
	Unit           string            `json:"unit"`
	Value          float64           `json:"value"`
}

// UpdateRateLimitsPolicyRequest represents the request to update a rate limits policy
type UpdateRateLimitsPolicyRequest struct {
	Name   string   `json:"name,omitempty"`
	Unit   string   `json:"unit,omitempty"`
	Value  *float64 `json:"value,omitempty"`
	Status string   `json:"status,omitempty"`
}

// CreateRateLimitsPolicyResponse represents the response from creating a rate limits policy
type CreateRateLimitsPolicyResponse struct {
	ID string `json:"id"`
}

// CreateRateLimitsPolicy creates a new rate limits policy
func (c *Client) CreateRateLimitsPolicy(ctx context.Context, req CreateRateLimitsPolicyRequest) (*CreateRateLimitsPolicyResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/policies/rate-limits", req)
	if err != nil {
		return nil, err
	}

	var response CreateRateLimitsPolicyResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetRateLimitsPolicy retrieves a rate limits policy by ID
func (c *Client) GetRateLimitsPolicy(ctx context.Context, id string) (*RateLimitsPolicy, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/policies/rate-limits/"+id, nil)
	if err != nil {
		return nil, err
	}

	var policy RateLimitsPolicy
	if err := json.Unmarshal(respBody, &policy); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &policy, nil
}

// ListRateLimitsPolicies retrieves all rate limits policies
func (c *Client) ListRateLimitsPolicies(ctx context.Context, workspaceID string) ([]RateLimitsPolicy, error) {
	path := "/policies/rate-limits"
	if workspaceID != "" {
		path = fmt.Sprintf("/policies/rate-limits?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []RateLimitsPolicy `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateRateLimitsPolicy updates a rate limits policy
func (c *Client) UpdateRateLimitsPolicy(ctx context.Context, id string, req UpdateRateLimitsPolicyRequest) (*RateLimitsPolicy, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/policies/rate-limits/"+id, req)
	if err != nil {
		return nil, err
	}

	return c.GetRateLimitsPolicy(ctx, id)
}

// DeleteRateLimitsPolicy deletes a rate limits policy
func (c *Client) DeleteRateLimitsPolicy(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/policies/rate-limits/"+id, nil)
	return err
}

// IntegrationWorkspaceUsageLimits represents usage limits for a workspace integration
type IntegrationWorkspaceUsageLimits struct {
	Type           string `json:"type,omitempty"`            // "cost" or "tokens"
	CreditLimit    *int   `json:"credit_limit,omitempty"`    // Credit limit value
	AlertThreshold *int   `json:"alert_threshold,omitempty"` // Alert threshold percentage
	PeriodicReset  string `json:"periodic_reset,omitempty"`  // "monthly" or "weekly"
}

// IntegrationWorkspaceRateLimits represents rate limits for a workspace integration
type IntegrationWorkspaceRateLimits struct {
	Type  string `json:"type,omitempty"`  // "requests" or "tokens"
	Unit  string `json:"unit,omitempty"`  // "rpd", "rph", "rpm"
	Value *int   `json:"value,omitempty"` // Limit value
}

// IntegrationWorkspace represents workspace access configuration for an integration
type IntegrationWorkspace struct {
	ID          string                            `json:"id"`
	Enabled     bool                              `json:"enabled"`
	UsageLimits []IntegrationWorkspaceUsageLimits `json:"usage_limits,omitempty"`
	RateLimits  []IntegrationWorkspaceRateLimits  `json:"rate_limits,omitempty"`
}

// IntegrationWorkspacesResponse represents the response from listing integration workspaces
type IntegrationWorkspacesResponse struct {
	Total      int                    `json:"total"`
	Workspaces []IntegrationWorkspace `json:"workspaces"`
}

// WorkspaceUpdateRequest represents a single workspace update in a bulk request
type WorkspaceUpdateRequest struct {
	ID          string                            `json:"id"`
	Enabled     bool                              `json:"enabled"`
	UsageLimits []IntegrationWorkspaceUsageLimits `json:"usage_limits,omitempty"`
	RateLimits  []IntegrationWorkspaceRateLimits  `json:"rate_limits,omitempty"`
	ResetUsage  bool                              `json:"reset_usage,omitempty"`
}

// GlobalWorkspaceAccess represents global workspace access settings
type GlobalWorkspaceAccess struct {
	Enabled     bool                              `json:"enabled"`
	UsageLimits []IntegrationWorkspaceUsageLimits `json:"usage_limits,omitempty"`
	RateLimits  []IntegrationWorkspaceRateLimits  `json:"rate_limits,omitempty"`
}

// BulkUpdateWorkspacesRequest represents the request to bulk update workspace access
type BulkUpdateWorkspacesRequest struct {
	Workspaces                      []WorkspaceUpdateRequest `json:"workspaces,omitempty"`
	GlobalWorkspaceAccess           *GlobalWorkspaceAccess   `json:"global_workspace_access,omitempty"`
	OverrideExistingWorkspaceAccess bool                     `json:"override_existing_workspace_access,omitempty"`
}

// GetIntegrationWorkspaces retrieves workspace access for an integration
func (c *Client) GetIntegrationWorkspaces(ctx context.Context, integrationSlug string) (*IntegrationWorkspacesResponse, error) {
	path := fmt.Sprintf("/integrations/%s/workspaces", integrationSlug)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response IntegrationWorkspacesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetIntegrationWorkspace retrieves a specific workspace's access for an integration
func (c *Client) GetIntegrationWorkspace(ctx context.Context, integrationSlug, workspaceID string) (*IntegrationWorkspace, error) {
	workspaces, err := c.GetIntegrationWorkspaces(ctx, integrationSlug)
	if err != nil {
		return nil, err
	}

	for _, ws := range workspaces.Workspaces {
		if ws.ID == workspaceID {
			return &ws, nil
		}
	}

	return nil, fmt.Errorf("workspace %s not found for integration %s", workspaceID, integrationSlug)
}

// UpdateIntegrationWorkspaces bulk updates workspace access for an integration
func (c *Client) UpdateIntegrationWorkspaces(ctx context.Context, integrationSlug string, req BulkUpdateWorkspacesRequest) error {
	path := fmt.Sprintf("/integrations/%s/workspaces", integrationSlug)
	_, err := c.doRequest(ctx, http.MethodPut, path, req)
	return err
}

// UpdateIntegrationWorkspace updates a single workspace's access for an integration
func (c *Client) UpdateIntegrationWorkspace(ctx context.Context, integrationSlug string, workspace WorkspaceUpdateRequest) error {
	req := BulkUpdateWorkspacesRequest{
		Workspaces: []WorkspaceUpdateRequest{workspace},
	}
	return c.UpdateIntegrationWorkspaces(ctx, integrationSlug, req)
}

// TokenPrice represents a token price configuration
type TokenPrice struct {
	Price float64 `json:"price"`
}

// PayAsYouGoPricing represents pay-as-you-go pricing configuration
type PayAsYouGoPricing struct {
	RequestToken  *TokenPrice `json:"request_token,omitempty"`
	ResponseToken *TokenPrice `json:"response_token,omitempty"`
}

// ModelPricingConfig represents pricing configuration for a model
type ModelPricingConfig struct {
	Type       string             `json:"type"` // "static" or other pricing types
	PayAsYouGo *PayAsYouGoPricing `json:"pay_as_you_go,omitempty"`
}

// IntegrationModel represents a model available through an integration
type IntegrationModel struct {
	Slug          string              `json:"slug"`
	Enabled       bool                `json:"enabled"`
	IsCustom      bool                `json:"is_custom,omitempty"`
	IsFinetune    bool                `json:"is_finetune,omitempty"`
	BaseModelSlug string              `json:"base_model_slug,omitempty"`
	PricingConfig *ModelPricingConfig `json:"pricing_config,omitempty"`
}

// IntegrationModelsResponse represents the response from listing integration models
type IntegrationModelsResponse struct {
	AllowAllModels bool               `json:"allow_all_models"`
	Models         []IntegrationModel `json:"models"`
}

// BulkUpdateModelsRequest represents the request to bulk update model access
type BulkUpdateModelsRequest struct {
	AllowAllModels *bool              `json:"allow_all_models,omitempty"`
	Models         []IntegrationModel `json:"models"`
}

// DeleteModelsRequest represents the request to delete custom models
type DeleteModelsRequest struct {
	Models []string `json:"models"`
}

// GetIntegrationModels retrieves all models for an integration
func (c *Client) GetIntegrationModels(ctx context.Context, integrationSlug string) (*IntegrationModelsResponse, error) {
	path := fmt.Sprintf("/integrations/%s/models", integrationSlug)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response IntegrationModelsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetIntegrationModel retrieves a specific model's access for an integration
func (c *Client) GetIntegrationModel(ctx context.Context, integrationSlug, modelSlug string) (*IntegrationModel, error) {
	models, err := c.GetIntegrationModels(ctx, integrationSlug)
	if err != nil {
		return nil, err
	}

	for _, m := range models.Models {
		if m.Slug == modelSlug {
			return &m, nil
		}
	}

	return nil, fmt.Errorf("model %s not found for integration %s", modelSlug, integrationSlug)
}

// UpdateIntegrationModels bulk updates model access for an integration
func (c *Client) UpdateIntegrationModels(ctx context.Context, integrationSlug string, req BulkUpdateModelsRequest) error {
	path := fmt.Sprintf("/integrations/%s/models", integrationSlug)
	_, err := c.doRequest(ctx, http.MethodPut, path, req)
	return err
}

// UpdateIntegrationModel updates a single model's access for an integration using GET-modify-PUT pattern
func (c *Client) UpdateIntegrationModel(ctx context.Context, integrationSlug string, model IntegrationModel) error {
	req := BulkUpdateModelsRequest{
		Models: []IntegrationModel{model},
	}
	return c.UpdateIntegrationModels(ctx, integrationSlug, req)
}

// DeleteIntegrationModels deletes custom models from an integration
func (c *Client) DeleteIntegrationModels(ctx context.Context, integrationSlug string, modelSlugs []string) error {
	path := fmt.Sprintf("/integrations/%s/models", integrationSlug)
	req := DeleteModelsRequest{
		Models: modelSlugs,
	}
	_, err := c.doRequest(ctx, http.MethodDelete, path, req)
	return err
}

// ============================================================================
// Prompt Collections
// ============================================================================

// PromptCollection represents a Portkey prompt collection
type PromptCollection struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	WorkspaceID        string `json:"workspace_id"`
	Slug               string `json:"slug,omitempty"`
	ParentCollectionID string `json:"parent_collection_id,omitempty"`
	IsDefault          int    `json:"is_default"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	LastUpdatedAt      string `json:"last_updated_at"`
}

// CreatePromptCollectionRequest represents the request to create a prompt collection
type CreatePromptCollectionRequest struct {
	Name               string `json:"name"`
	WorkspaceID        string `json:"workspace_id"`
	ParentCollectionID string `json:"parent_collection_id,omitempty"`
}

// CreatePromptCollectionResponse represents the response from creating a prompt collection
type CreatePromptCollectionResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// UpdatePromptCollectionRequest represents the request to update a prompt collection
type UpdatePromptCollectionRequest struct {
	Name string `json:"name,omitempty"`
}

// CreatePromptCollection creates a new prompt collection
func (c *Client) CreatePromptCollection(ctx context.Context, req CreatePromptCollectionRequest) (*CreatePromptCollectionResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/collections", req)
	if err != nil {
		return nil, err
	}

	var response CreatePromptCollectionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetPromptCollection retrieves a prompt collection by ID
func (c *Client) GetPromptCollection(ctx context.Context, id string) (*PromptCollection, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/collections/"+id, nil)
	if err != nil {
		return nil, err
	}

	var collection PromptCollection
	if err := json.Unmarshal(respBody, &collection); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &collection, nil
}

// ListPromptCollections retrieves all prompt collections, optionally filtered by workspace
func (c *Client) ListPromptCollections(ctx context.Context, workspaceID string) ([]PromptCollection, error) {
	path := "/collections"
	if workspaceID != "" {
		path = fmt.Sprintf("/collections?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data  []PromptCollection `json:"data"`
		Total int                `json:"total"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdatePromptCollection updates a prompt collection
func (c *Client) UpdatePromptCollection(ctx context.Context, id string, req UpdatePromptCollectionRequest) (*PromptCollection, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/collections/"+id, req)
	if err != nil {
		return nil, err
	}

	// Fetch updated collection details since update may return empty response
	collection, err := c.GetPromptCollection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("collection updated but failed to retrieve details: %w", err)
	}

	return collection, nil
}

// DeletePromptCollection deletes a prompt collection
func (c *Client) DeletePromptCollection(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/collections/"+id, nil)
	return err
}

// ============================================================================
// MCP Integrations
// ============================================================================

// McpIntegration represents a Portkey MCP integration (org-level MCP server registration)
type McpIntegration struct {
	ID                    string `json:"id"`
	Slug                  string `json:"slug"`
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	URL                   string `json:"url"`
	AuthType              string `json:"auth_type"`
	Transport             string `json:"transport"`
	Configurations        any    `json:"configurations,omitempty"`
	WorkspaceID           string `json:"workspace_id,omitempty"`
	Type                  string `json:"type,omitempty"`
	Status                string `json:"status,omitempty"`
	OwnerID               string `json:"owner_id,omitempty"`
	CreatedAt             string `json:"created_at,omitempty"`
	LastUpdatedAt         string `json:"last_updated_at,omitempty"`
	GlobalWorkspaceAccess any    `json:"global_workspace_access,omitempty"`
}

// CreateMcpIntegrationRequest represents the request to create an MCP integration
type CreateMcpIntegrationRequest struct {
	Name           string `json:"name"`
	Slug           string `json:"slug,omitempty"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url"`
	AuthType       string `json:"auth_type"`
	Transport      string `json:"transport"`
	Configurations any    `json:"configurations,omitempty"`
	WorkspaceID    string `json:"workspace_id,omitempty"`
}

// CreateMcpIntegrationResponse represents the response from creating an MCP integration
type CreateMcpIntegrationResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

// UpdateMcpIntegrationRequest represents the request to update an MCP integration
type UpdateMcpIntegrationRequest struct {
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url,omitempty"`
	AuthType       string `json:"auth_type,omitempty"`
	Transport      string `json:"transport,omitempty"`
	Configurations any    `json:"configurations,omitempty"`
}

// CreateMcpIntegration creates a new MCP integration
func (c *Client) CreateMcpIntegration(ctx context.Context, req CreateMcpIntegrationRequest) (*CreateMcpIntegrationResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/mcp-integrations", req)
	if err != nil {
		return nil, err
	}

	var response CreateMcpIntegrationResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// GetMcpIntegration retrieves an MCP integration by ID or slug
func (c *Client) GetMcpIntegration(ctx context.Context, id string) (*McpIntegration, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/mcp-integrations/"+id, nil)
	if err != nil {
		return nil, err
	}

	var integration McpIntegration
	if err := json.Unmarshal(respBody, &integration); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &integration, nil
}

// ListMcpIntegrations retrieves all MCP integrations, optionally filtered by workspace
func (c *Client) ListMcpIntegrations(ctx context.Context, workspaceID string) ([]McpIntegration, error) {
	path := "/mcp-integrations"
	if workspaceID != "" {
		path = fmt.Sprintf("/mcp-integrations?workspace_id=%s", workspaceID)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []McpIntegration `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateMcpIntegration updates an MCP integration
func (c *Client) UpdateMcpIntegration(ctx context.Context, id string, req UpdateMcpIntegrationRequest) (*McpIntegration, error) {
	_, err := c.doRequest(ctx, http.MethodPut, "/mcp-integrations/"+id, req)
	if err != nil {
		return nil, err
	}

	return c.GetMcpIntegration(ctx, id)
}

// DeleteMcpIntegration deletes an MCP integration
func (c *Client) DeleteMcpIntegration(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/mcp-integrations/"+id, nil)
	return err
}

// ============================================================================
// MCP Capabilities (shared types for integration capabilities)
// ============================================================================

// McpCapability represents a capability (tool/resource/prompt) in an MCP integration or server
type McpCapability struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

// McpCapabilitiesUpdateRequest represents the request to update capabilities
type McpCapabilitiesUpdateRequest struct {
	Capabilities []McpCapability `json:"capabilities"`
}

// GetMcpIntegrationCapabilities retrieves capabilities for an MCP integration
func (c *Client) GetMcpIntegrationCapabilities(ctx context.Context, id string) ([]McpCapability, error) {
	path := fmt.Sprintf("/mcp-integrations/%s/capabilities", id)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []McpCapability `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Data, nil
}

// UpdateMcpIntegrationCapabilities updates capabilities for an MCP integration
func (c *Client) UpdateMcpIntegrationCapabilities(ctx context.Context, id string, capabilities []McpCapability) error {
	path := fmt.Sprintf("/mcp-integrations/%s/capabilities", id)
	req := McpCapabilitiesUpdateRequest{Capabilities: capabilities}
	_, err := c.doRequest(ctx, http.MethodPut, path, req)
	return err
}

// ============================================================================
// MCP Integration Workspace Access
// ============================================================================

// McpIntegrationWorkspace represents workspace access for an MCP integration
type McpIntegrationWorkspace struct {
	WorkspaceID string `json:"id"`
	Enabled     bool   `json:"enabled"`
}

// McpIntegrationWorkspacesResponse represents the response from listing MCP integration workspaces
type McpIntegrationWorkspacesResponse struct {
	Workspaces []McpIntegrationWorkspace `json:"workspaces"`
}

// McpIntegrationWorkspaceUpdate represents an update to workspace access
type McpIntegrationWorkspaceUpdate struct {
	WorkspaceID string `json:"id"`
	Enabled     bool   `json:"enabled"`
}

// McpIntegrationWorkspacesUpdateRequest represents the request to update workspace access
type McpIntegrationWorkspacesUpdateRequest struct {
	Workspaces []McpIntegrationWorkspaceUpdate `json:"workspaces"`
}

// GetMcpIntegrationWorkspaces retrieves workspace access for an MCP integration
func (c *Client) GetMcpIntegrationWorkspaces(ctx context.Context, id string) ([]McpIntegrationWorkspace, error) {
	path := fmt.Sprintf("/mcp-integrations/%s/workspaces", id)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response McpIntegrationWorkspacesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return response.Workspaces, nil
}

// GetMcpIntegrationWorkspace retrieves a specific workspace's access for an MCP integration
func (c *Client) GetMcpIntegrationWorkspace(ctx context.Context, id, workspaceID string) (*McpIntegrationWorkspace, error) {
	workspaces, err := c.GetMcpIntegrationWorkspaces(ctx, id)
	if err != nil {
		return nil, err
	}

	for _, ws := range workspaces {
		if ws.WorkspaceID == workspaceID {
			return &ws, nil
		}
	}

	return nil, fmt.Errorf("workspace %s not found for MCP integration %s", workspaceID, id)
}

// UpdateMcpIntegrationWorkspace updates a single workspace's access for an MCP integration
func (c *Client) UpdateMcpIntegrationWorkspace(ctx context.Context, id string, update McpIntegrationWorkspaceUpdate) error {
	path := fmt.Sprintf("/mcp-integrations/%s/workspaces", id)
	req := McpIntegrationWorkspacesUpdateRequest{
		Workspaces: []McpIntegrationWorkspaceUpdate{update},
	}
	_, err := c.doRequest(ctx, http.MethodPut, path, req)
	return err
}

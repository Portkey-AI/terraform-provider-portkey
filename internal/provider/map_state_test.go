package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/portkey-ai/terraform-provider-portkey/internal/client"
)

// --- mapPartialToState tests ---

func TestMapPartialToState_ExternalChangeDetected(t *testing.T) {
	r := &promptPartialResource{}

	state := &promptPartialResourceModel{
		Content:                types.StringValue("terraform content"),
		Version:                types.Int64Value(2),
		PromptPartialVersionID: types.StringValue("old-version-id"),
	}

	partial := &client.PromptPartial{
		ID:                     "id-1",
		Slug:                   "my-partial",
		Name:                   "My Partial",
		String:                 "console-edited content",
		Status:                 "active",
		Version:                3,
		PromptPartialVersionID: "new-version-id",
		CreatedAt:              time.Now(),
	}

	r.mapPartialToState(state, partial)

	if state.Content.ValueString() != "console-edited content" {
		t.Errorf("expected content to be refreshed from API, got %q", state.Content.ValueString())
	}
	if state.Version.ValueInt64() != 3 {
		t.Errorf("expected version 3, got %d", state.Version.ValueInt64())
	}
	if state.PromptPartialVersionID.ValueString() != "new-version-id" {
		t.Errorf("expected new version ID, got %q", state.PromptPartialVersionID.ValueString())
	}
}

func TestMapPartialToState_NoExternalChange(t *testing.T) {
	r := &promptPartialResource{}

	state := &promptPartialResourceModel{
		Content:                types.StringValue("terraform content"),
		Version:                types.Int64Value(2),
		PromptPartialVersionID: types.StringValue("current-version-id"),
	}

	// API returns same version — no external change, content preserved from state
	partial := &client.PromptPartial{
		ID:                     "id-1",
		Slug:                   "my-partial",
		Name:                   "My Partial",
		String:                 "stale api content",
		Status:                 "active",
		Version:                2,
		PromptPartialVersionID: "current-version-id",
		CreatedAt:              time.Now(),
	}

	r.mapPartialToState(state, partial)

	if state.Content.ValueString() != "terraform content" {
		t.Errorf("expected content to be preserved from state, got %q", state.Content.ValueString())
	}
}

func TestMapPartialToState_RollbackDetected(t *testing.T) {
	r := &promptPartialResource{}

	// State at version 3
	state := &promptPartialResourceModel{
		Content:                types.StringValue("terraform content"),
		Version:                types.Int64Value(3),
		PromptPartialVersionID: types.StringValue("version-id-3"),
	}

	// API returns version 1 — someone rolled back in console
	partial := &client.PromptPartial{
		ID:                     "id-1",
		Slug:                   "my-partial",
		Name:                   "My Partial",
		String:                 "rolled-back content",
		Status:                 "active",
		Version:                1,
		PromptPartialVersionID: "version-id-1",
		CreatedAt:              time.Now(),
	}

	r.mapPartialToState(state, partial)

	if state.Content.ValueString() != "rolled-back content" {
		t.Errorf("expected content to be refreshed from API on rollback, got %q", state.Content.ValueString())
	}
	if state.Version.ValueInt64() != 1 {
		t.Errorf("expected version 1, got %d", state.Version.ValueInt64())
	}
}

func TestMapPartialToState_FirstPopulation(t *testing.T) {
	r := &promptPartialResource{}

	// Fresh state (create or import) — version is null
	state := &promptPartialResourceModel{}

	partial := &client.PromptPartial{
		ID:                     "id-1",
		Slug:                   "my-partial",
		Name:                   "My Partial",
		String:                 "api content",
		Status:                 "active",
		Version:                1,
		PromptPartialVersionID: "version-id-1",
		CreatedAt:              time.Now(),
	}

	r.mapPartialToState(state, partial)

	if state.Content.ValueString() != "api content" {
		t.Errorf("expected content from API on first population, got %q", state.Content.ValueString())
	}
	if state.Version.ValueInt64() != 1 {
		t.Errorf("expected version 1, got %d", state.Version.ValueInt64())
	}
}

func TestMapPartialToState_VersionDescriptionNotImported(t *testing.T) {
	r := &promptPartialResource{}

	// State has no version_description (user didn't set it)
	state := &promptPartialResourceModel{
		Content: types.StringValue("terraform content"),
		Version: types.Int64Value(1),
	}

	// API returns a version_description from a console edit
	partial := &client.PromptPartial{
		ID:                 "id-1",
		Slug:               "my-partial",
		Name:               "My Partial",
		String:             "terraform content",
		Status:             "active",
		Version:            1,
		VersionDescription: "set via console",
		CreatedAt:          time.Now(),
	}

	r.mapPartialToState(state, partial)

	if !state.VersionDescription.IsNull() {
		t.Errorf("expected version_description to remain null, got %q", state.VersionDescription.ValueString())
	}
}

// --- mapPromptToState tests ---

func TestMapPromptToState_ExternalChangeDetected(t *testing.T) {
	r := &promptResource{}

	state := &promptResourceModel{
		Template:      types.StringValue("terraform template"),
		Model:         types.StringValue("gpt-4"),
		PromptVersion: types.Int64Value(1),
		Parameters:    types.StringValue(`{"model":"gpt-4"}`),
	}

	// API returns version 2 — someone edited in console
	prompt := &client.Prompt{
		ID:                  "id-1",
		Slug:                "my-prompt",
		Name:                "My Prompt",
		String:              "console-edited template",
		Model:               "gpt-5-mini",
		Status:              "active",
		PromptVersion:       2,
		PromptVersionID:     "version-id-2",
		PromptVersionStatus: "active",
		CreatedAt:           time.Now(),
	}

	r.mapPromptToState(state, prompt)

	if state.Template.ValueString() != "console-edited template" {
		t.Errorf("expected template to be refreshed from API, got %q", state.Template.ValueString())
	}
	if state.Model.ValueString() != "gpt-5-mini" {
		t.Errorf("expected model to be refreshed from API, got %q", state.Model.ValueString())
	}
	if state.PromptVersion.ValueInt64() != 2 {
		t.Errorf("expected version 2, got %d", state.PromptVersion.ValueInt64())
	}
}

func TestMapPromptToState_NoExternalChange(t *testing.T) {
	r := &promptResource{}

	state := &promptResourceModel{
		Template:        types.StringValue("terraform template"),
		Model:           types.StringValue("gpt-4"),
		PromptVersion:   types.Int64Value(1),
		PromptVersionID: types.StringValue("version-id-1"),
		Parameters:      types.StringValue(`{"model":"gpt-4"}`),
	}

	// API returns same version — content preserved from state
	prompt := &client.Prompt{
		ID:                  "id-1",
		Slug:                "my-prompt",
		Name:                "My Prompt",
		String:              "stale api template",
		Model:               "stale-model",
		Status:              "active",
		PromptVersion:       1,
		PromptVersionID:     "version-id-1",
		PromptVersionStatus: "active",
		CreatedAt:           time.Now(),
	}

	r.mapPromptToState(state, prompt)

	if state.Template.ValueString() != "terraform template" {
		t.Errorf("expected template to be preserved from state, got %q", state.Template.ValueString())
	}
	if state.Model.ValueString() != "gpt-4" {
		t.Errorf("expected model to be preserved from state, got %q", state.Model.ValueString())
	}
}

func TestMapPromptToState_RollbackDetected(t *testing.T) {
	r := &promptResource{}

	// State at version 3
	state := &promptResourceModel{
		Template:      types.StringValue("terraform template"),
		Model:         types.StringValue("gpt-4"),
		PromptVersion: types.Int64Value(3),
		Parameters:    types.StringValue(`{"model":"gpt-4"}`),
	}

	// API returns version 1 — someone rolled back in console
	prompt := &client.Prompt{
		ID:                  "id-1",
		Slug:                "my-prompt",
		Name:                "My Prompt",
		String:              "rolled-back template",
		Model:               "gpt-3.5-turbo",
		Status:              "active",
		PromptVersion:       1,
		PromptVersionID:     "version-id-1",
		PromptVersionStatus: "active",
		CreatedAt:           time.Now(),
	}

	r.mapPromptToState(state, prompt)

	if state.Template.ValueString() != "rolled-back template" {
		t.Errorf("expected template to be refreshed from API on rollback, got %q", state.Template.ValueString())
	}
	if state.Model.ValueString() != "gpt-3.5-turbo" {
		t.Errorf("expected model to be refreshed from API on rollback, got %q", state.Model.ValueString())
	}
	if state.PromptVersion.ValueInt64() != 1 {
		t.Errorf("expected version 1, got %d", state.PromptVersion.ValueInt64())
	}
}

func TestMapPromptToState_FirstPopulation(t *testing.T) {
	r := &promptResource{}

	// Fresh state (create or import)
	state := &promptResourceModel{}

	prompt := &client.Prompt{
		ID:                  "id-1",
		Slug:                "my-prompt",
		Name:                "My Prompt",
		String:              "api template",
		Model:               "gpt-4",
		Status:              "active",
		PromptVersion:       1,
		PromptVersionID:     "version-id-1",
		PromptVersionStatus: "active",
		CreatedAt:           time.Now(),
	}

	r.mapPromptToState(state, prompt)

	if state.Template.ValueString() != "api template" {
		t.Errorf("expected template from API on first population, got %q", state.Template.ValueString())
	}
	if state.PromptVersion.ValueInt64() != 1 {
		t.Errorf("expected version 1, got %d", state.PromptVersion.ValueInt64())
	}
}

func TestMapPromptToState_VersionDescriptionNotImported(t *testing.T) {
	r := &promptResource{}

	state := &promptResourceModel{
		Template:      types.StringValue("template"),
		Model:         types.StringValue("gpt-4"),
		PromptVersion: types.Int64Value(1),
		Parameters:    types.StringValue(`{"model":"gpt-4"}`),
	}

	prompt := &client.Prompt{
		ID:                       "id-1",
		Slug:                     "my-prompt",
		Name:                     "My Prompt",
		String:                   "template",
		Model:                    "gpt-4",
		Status:                   "active",
		PromptVersion:            1,
		PromptVersionID:          "version-id-1",
		PromptVersionStatus:      "active",
		PromptVersionDescription: "set via console",
		CreatedAt:                time.Now(),
	}

	r.mapPromptToState(state, prompt)

	if !state.VersionDescription.IsNull() {
		t.Errorf("expected version_description to remain null, got %q", state.VersionDescription.ValueString())
	}
}

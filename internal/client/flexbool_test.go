package client

import (
	"encoding/json"
	"testing"
)

func TestFlexBool_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		// JSON boolean
		{name: "bool true", input: `true`, want: true},
		{name: "bool false", input: `false`, want: false},

		// JSON number (Portkey API returns 1 for is_custom=true)
		{name: "number 1", input: `1`, want: true},
		{name: "number 0", input: `0`, want: false},
		{name: "number 1.0", input: `1.0`, want: true},
		{name: "number 0.0", input: `0.0`, want: false},

		// JSON string (Portkey API returns "false" for is_custom=false)
		{name: "string false", input: `"false"`, want: false},
		{name: "string true", input: `"true"`, want: true},
		{name: "string 0", input: `"0"`, want: false},
		{name: "string 1", input: `"1"`, want: true},
		{name: "string empty", input: `""`, want: false},

		// JSON null — treated as false (API won't send this, but be defensive)
		{name: "null", input: `null`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FlexBool
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bool(got) != tt.want {
				t.Errorf("got %v, want %v", bool(got), tt.want)
			}
		})
	}
}

func TestFlexBool_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		val  FlexBool
		want string
	}{
		{name: "true", val: FlexBool(true), want: "true"},
		{name: "false", val: FlexBool(false), want: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("got %s, want %s", string(data), tt.want)
			}
		})
	}
}

// TestIntegrationModel_UnmarshalRealAPIResponse tests unmarshaling a response
// that mimics the real Portkey API behaviour: is_custom as integer 1 and
// is_finetune as string "false".
func TestIntegrationModel_UnmarshalRealAPIResponse(t *testing.T) {
	// This is the exact shape that causes the original bug:
	// is_custom comes back as the number 1, is_finetune as the string "false".
	payload := `{
		"slug": "arn:aws:bedrock:us-east-1:123456789:inference-profile/test",
		"enabled": true,
		"is_custom": 1,
		"is_finetune": "false",
		"base_model_slug": "global.anthropic.claude-sonnet-4-6"
	}`

	var model IntegrationModel
	if err := json.Unmarshal([]byte(payload), &model); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if model.Slug != "arn:aws:bedrock:us-east-1:123456789:inference-profile/test" {
		t.Errorf("unexpected slug: %s", model.Slug)
	}
	if !model.Enabled {
		t.Error("expected enabled=true")
	}
	if !bool(model.IsCustom) {
		t.Error("expected is_custom=true (from integer 1)")
	}
	if bool(model.IsFinetune) {
		t.Error("expected is_finetune=false (from string \"false\")")
	}
	if model.BaseModelSlug != "global.anthropic.claude-sonnet-4-6" {
		t.Errorf("unexpected base_model_slug: %s", model.BaseModelSlug)
	}

	// Verify round-trip: marshal back to JSON and unmarshal again
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var model2 IntegrationModel
	if err := json.Unmarshal(data, &model2); err != nil {
		t.Fatalf("failed to round-trip unmarshal: %v", err)
	}
	if bool(model2.IsCustom) != bool(model.IsCustom) {
		t.Error("round-trip changed is_custom")
	}
	if bool(model2.IsFinetune) != bool(model.IsFinetune) {
		t.Error("round-trip changed is_finetune")
	}
}

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkspaceSlugAfterUpdate(t *testing.T) {
	cases := []struct {
		name    string
		planned types.String
		apiSlug string
		want    types.String
	}{
		{
			name:    "unknown plan is filled from the API",
			planned: types.StringUnknown(),
			apiSlug: "ws-exampl-abc123",
			want:    types.StringValue("ws-exampl-abc123"),
		},
		{
			name:    "unknown plan with no API slug becomes null",
			planned: types.StringUnknown(),
			apiSlug: "",
			want:    types.StringNull(),
		},
		{
			// State from an older provider version upgrades slug to null. With
			// UseStateForUnknown the plan is that null, so writing the API slug
			// would be an inconsistent result after apply (-refresh=false).
			name:    "null plan stays null",
			planned: types.StringNull(),
			apiSlug: "ws-exampl-abc123",
			want:    types.StringNull(),
		},
		{
			name:    "known plan is kept",
			planned: types.StringValue("ws-exampl-abc123"),
			apiSlug: "ws-renamed-def456",
			want:    types.StringValue("ws-exampl-abc123"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workspaceSlugAfterUpdate(tc.planned, tc.apiSlug); !got.Equal(tc.want) {
				t.Fatalf("workspaceSlugAfterUpdate(%s, %q) = %s, want %s", tc.planned, tc.apiSlug, got, tc.want)
			}
		})
	}
}

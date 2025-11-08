package qlab

import (
	"testing"

	"github.com/zenibako/qlab-golang/messages"
)

func TestGetAddress(t *testing.T) {
	ws := Workspace{
		workspace_id:   "TEST-WORKSPACE-ID",
		addressBuilder: messages.NewOSCAddressBuilder("TEST-WORKSPACE-ID"),
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "application level - connect",
			input:    "/connect",
			expected: "/connect",
		},
		{
			name:     "application level - updates",
			input:    "/updates",
			expected: "/updates",
		},
		{
			name:     "application level - udpReplyPort",
			input:    "/udpReplyPort",
			expected: "/udpReplyPort",
		},
		{
			name:     "application level - alwaysReply",
			input:    "/alwaysReply",
			expected: "/alwaysReply",
		},
		{
			name:     "workspace level - select",
			input:    "/select/1",
			expected: "/workspace/TEST-WORKSPACE-ID/select/1",
		},
		{
			name:     "workspace level - go",
			input:    "/go",
			expected: "/workspace/TEST-WORKSPACE-ID/go",
		},
		{
			name:     "workspace level - panic",
			input:    "/panic",
			expected: "/workspace/TEST-WORKSPACE-ID/panic",
		},
		{
			name:     "already has workspace prefix",
			input:    "/workspace/OTHER-ID/select/1",
			expected: "/workspace/OTHER-ID/select/1",
		},
		{
			name:     "non-slash prefix",
			input:    "select/1",
			expected: "select/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ws.GetAddress(tt.input)
			if result != tt.expected {
				t.Errorf("GetAddress(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetAddressWithoutWorkspaceID(t *testing.T) {
	ws := Workspace{
		workspace_id:   "",
		addressBuilder: messages.NewOSCAddressBuilder(""),
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no workspace ID - returns as-is",
			input:    "/select/1",
			expected: "/select/1",
		},
		{
			name:     "application level still works",
			input:    "/connect",
			expected: "/connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ws.GetAddress(tt.input)
			if result != tt.expected {
				t.Errorf("GetAddress(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

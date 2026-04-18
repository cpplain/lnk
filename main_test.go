package main

import (
	"testing"
)

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
		wantRemain  []string
	}{
		{
			name:        "simple command",
			args:        []string{"create", "."},
			wantCommand: "create",
			wantRemain:  []string{"."},
		},
		{
			name:        "flags before command",
			args:        []string{"--dry-run", "create", "."},
			wantCommand: "create",
			wantRemain:  []string{"--dry-run", "."},
		},
		{
			name:        "ignore space value before command",
			args:        []string{"--ignore", "pattern", "create", "."},
			wantCommand: "create",
			wantRemain:  []string{"--ignore", "pattern", "."},
		},
		{
			name:        "ignore equals value before command",
			args:        []string{"--ignore=pattern", "create", "."},
			wantCommand: "create",
			wantRemain:  []string{"--ignore=pattern", "."},
		},
		{
			name:        "ignore with no command",
			args:        []string{"--ignore", "pattern"},
			wantCommand: "",
			wantRemain:  []string{"--ignore", "pattern"},
		},
		{
			name:        "multiple ignore before command",
			args:        []string{"--ignore", "*.swp", "--ignore", "*.bak", "create", "."},
			wantCommand: "create",
			wantRemain:  []string{"--ignore", "*.swp", "--ignore", "*.bak", "."},
		},
		{
			name:        "ignore after command",
			args:        []string{"create", "--ignore", "pattern", "."},
			wantCommand: "create",
			wantRemain:  []string{"--ignore", "pattern", "."},
		},
		{
			name:        "no args",
			args:        []string{},
			wantCommand: "",
			wantRemain:  []string{},
		},
		{
			name:        "only flags",
			args:        []string{"--dry-run", "--verbose"},
			wantCommand: "",
			wantRemain:  []string{"--dry-run", "--verbose"},
		},
		{
			name:        "double dash stops parsing",
			args:        []string{"--", "create", "."},
			wantCommand: "",
			wantRemain:  []string{"--", "create", "."},
		},
		{
			name:        "ignore is last flag with no value following",
			args:        []string{"--ignore"},
			wantCommand: "",
			wantRemain:  []string{"--ignore"},
		},
		{
			name:        "ignore value starts with dash is not skipped",
			args:        []string{"--ignore", "--dry-run", "create", "."},
			wantCommand: "create",
			wantRemain:  []string{"--ignore", "--dry-run", "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotRemain := extractCommand(tt.args)
			if gotCmd != tt.wantCommand {
				t.Errorf("extractCommand(%v) command = %q, want %q", tt.args, gotCmd, tt.wantCommand)
			}
			if len(gotRemain) != len(tt.wantRemain) {
				t.Errorf("extractCommand(%v) remaining = %v (len %d), want %v (len %d)",
					tt.args, gotRemain, len(gotRemain), tt.wantRemain, len(tt.wantRemain))
				return
			}
			for i := range gotRemain {
				if gotRemain[i] != tt.wantRemain[i] {
					t.Errorf("extractCommand(%v) remaining[%d] = %q, want %q",
						tt.args, i, gotRemain[i], tt.wantRemain[i])
				}
			}
		})
	}
}

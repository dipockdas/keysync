package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveEnvironment_NoProject(t *testing.T) {
	project = ""
	envFlag = "production"
	cmd := &cobra.Command{Use: "push"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")
	_ = cmd.Flags().Set("env", "production")

	got := resolveEnvironmentForCommand(cmd)
	if got != "" {
		t.Errorf("resolveEnvironmentForCommand = %q, want empty when no project", got)
	}
}

func TestResolveEnvironment_ProjectWideWhenEnvOmitted(t *testing.T) {
	project = "my-app"
	envFlag = ""
	cmd := &cobra.Command{Use: "push"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")

	got := resolveEnvironmentForCommand(cmd)
	if got != "" {
		t.Errorf("resolveEnvironmentForCommand = %q, want empty for project-wide scope", got)
	}
}

func TestResolveEnvironment_GetDoesNotDefaultEnv(t *testing.T) {
	project = "my-app"
	envFlag = ""
	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")

	if got := resolveEnvironmentForCommand(cmd); got != "" {
		t.Errorf("resolveEnvironmentForCommand = %q, want empty for get", got)
	}
	if got := envForGet(cmd); got != "" {
		t.Errorf("envForGet = %q, want empty when --env omitted", got)
	}
}

func TestResolveEnvironment_ExplicitEnv(t *testing.T) {
	project = "my-app"
	envFlag = "production"
	cmd := &cobra.Command{Use: "push"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")
	_ = cmd.Flags().Set("env", "production")

	got := resolveEnvironmentForCommand(cmd)
	if got != "production" {
		t.Errorf("resolveEnvironmentForCommand = %q, want production", got)
	}
}

func TestResolveEnvironment_ExplicitDev(t *testing.T) {
	project = "my-app"
	envFlag = "dev"
	cmd := &cobra.Command{Use: "set"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")
	_ = cmd.Flags().Set("env", "dev")

	got := resolveEnvironmentForCommand(cmd)
	if got != "dev" {
		t.Errorf("resolveEnvironmentForCommand = %q, want dev", got)
	}
}

func TestResolveEnvironment_ExplicitEmptyProjectScope(t *testing.T) {
	project = "my-app"
	envFlag = ""
	cmd := &cobra.Command{Use: "set"}
	cmd.Flags().StringVar(&envFlag, "env", "", "")
	_ = cmd.Flags().Set("env", "")

	got := resolveEnvironmentForCommand(cmd)
	if got != "" {
		t.Errorf("resolveEnvironmentForCommand = %q, want empty for project-wide scope", got)
	}
}

func TestResolveEnvironmentFromArgs_NoEnvMeansProjectWide(t *testing.T) {
	project = "my-app"
	got := resolveEnvironmentFromArgs("push", nil)
	if got != "" {
		t.Errorf("resolveEnvironmentFromArgs(push) = %q, want empty", got)
	}
}

func TestResolveEnvironmentFromArgs_ExplicitEnvInArgs(t *testing.T) {
	project = "my-app"
	got := resolveEnvironmentFromArgs("push", []string{"--env", "staging"})
	if got != "staging" {
		t.Errorf("resolveEnvironmentFromArgs = %q, want staging", got)
	}
}

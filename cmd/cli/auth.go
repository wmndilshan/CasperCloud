package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with the API",
	}
	cmd.AddCommand(authLoginCmd())
	return cmd
}

func authLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in with email and password; saves JWT and project to ~/.casper/config.json",
		RunE:  runAuthLogin,
	}
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	base, err := apiBaseURL()
	if err != nil {
		return err
	}
	client, err := NewClient(ClientOptions{BaseURL: base})
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, "Email: ")
	email, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	email = strings.TrimSpace(email)

	fmt.Fprint(os.Stderr, "Password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr)
	password := string(pwBytes)

	var loginOut struct {
		Data struct {
			Token           string          `json:"token"`
			User            repository.User `json:"user"`
			ActiveProjectID *uuid.UUID      `json:"active_project_id,omitempty"`
		} `json:"data"`
	}
	loginBody := apitypes.LoginRequest{
		Email:    email,
		Password: password,
	}
	if err := client.do("POST", "/v1/auth/login", loginBody, false, &loginOut); err != nil {
		return err
	}
	resp := loginOut.Data

	client.Token = resp.Token

	var projectID uuid.UUID
	var token string
	if resp.ActiveProjectID != nil && *resp.ActiveProjectID != uuid.Nil {
		projectID = *resp.ActiveProjectID
		token = resp.Token
	} else {
		var list struct {
			Data []repository.Project `json:"data"`
		}
		if err := client.do("GET", "/v1/projects", nil, true, &list); err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		if len(list.Data) == 0 {
			return fmt.Errorf("login ok but no projects found; create one with POST /v1/projects then run login again with project scope")
		}
		projectID = list.Data[0].ID
		var switchOut struct {
			Data struct {
				Token           string          `json:"token"`
				User            repository.User `json:"user"`
				ActiveProjectID *uuid.UUID      `json:"active_project_id,omitempty"`
			} `json:"data"`
		}
		sw := apitypes.SwitchProjectRequest{ProjectID: projectID.String()}
		if err := client.do("POST", "/v1/auth/switch-project", sw, true, &switchOut); err != nil {
			return fmt.Errorf("switch project: %w", err)
		}
		r2 := switchOut.Data
		if r2.ActiveProjectID != nil {
			projectID = *r2.ActiveProjectID
		}
		token = r2.Token
		resp.User = r2.User
	}

	if err := writeMergedConfig(map[string]any{
		"api_base_url": base,
		"token":        token,
		"project_id":   projectID.String(),
	}); err != nil {
		return err
	}
	viper.Set("token", token)
	viper.Set("project_id", projectID.String())
	viper.Set("api_base_url", base)

	fmt.Printf("Logged in as %s (project %s)\n", resp.User.Email, projectID)
	return nil
}

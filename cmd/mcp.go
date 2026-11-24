package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	platmodel "github.com/codefresh-io/go-sdk/pkg/model/platform"
	"github.com/codefresh-io/go-sdk/pkg/rest"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	cfutils "github.com/mniak/cfutils"
	"github.com/spf13/cobra"
)

func newMCPCommand() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP server exposing Codefresh API over stdio",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			setupMCPLogging(verbose)
			slog.Debug("starting MCP server")

			mcpServer := server.NewMCPServer("cfutils", "0.1.0",
				server.WithInstructions("Codefresh utilities exposed over MCP. Tools return structured JSON."),
				server.WithToolCapabilities(false),
				server.WithResourceCapabilities(false, false),
				server.WithPromptCapabilities(false),
				server.WithRecovery(),
			)

			registerPipelines(mcpServer)
			registerBuilds(mcpServer)
			registerWorkflows(mcpServer)
			registerAccounts(mcpServer)
			registerRuntimes(mcpServer)
			registerClusters(mcpServer)
			registerGitSources(mcpServer)

			if err := server.ServeStdio(mcpServer); err != nil {
				slog.Error("server terminated", "error", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Log debug messages to stderr")

	return cmd
}

func setupMCPLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

func newClient() (*cfutils.Client, error) {
	browser := flags.Browser
	accessToken, _, err := cfutils.TokensFromBrowser(browser)
	if err != nil {
		return nil, fmt.Errorf("reading tokens from %s: %w", browser, err)
	}
	return cfutils.New("", accessToken), nil
}

// --- pipelines ---

func registerPipelines(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_pipeline_list",
			mcp.WithDescription("List Codefresh pipelines, optionally filtered by name"),
			mcp.WithInputSchema[PipelineListInput](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, in PipelineListInput) ([]PipelineItem, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			filters := map[string]string{"limit": "50"}
			if in.Name != "" {
				filters["id"] = in.Name
			}
			pipelines, err := client.Rest().Pipeline().List(filters)
			if err != nil {
				return nil, fmt.Errorf("listing pipelines: %w", err)
			}
			items := make([]PipelineItem, len(pipelines))
			for i, p := range pipelines {
				items[i] = PipelineItem{
					Name:    p.Metadata.Name,
					Project: p.Metadata.Project,
					ID:      p.Metadata.ID,
				}
			}
			return items, nil
		}),
	)

	s.AddTool(
		mcp.NewTool("codefresh_pipeline_run",
			mcp.WithDescription("Trigger a Codefresh pipeline build"),
			mcp.WithInputSchema[PipelineRunInput](),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, in PipelineRunInput) (*PipelineRunResult, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			buildID, err := client.Rest().Pipeline().Run(in.ID, &rest.RunOptions{
				Branch:    in.Branch,
				Variables: in.Variables,
			})
			if err != nil {
				return nil, fmt.Errorf("running pipeline: %w", err)
			}
			return &PipelineRunResult{BuildID: buildID}, nil
		}),
	)
}

type PipelineListInput struct {
	Name string `json:"name" jsonschema:"Filter pipelines by name (loose match)"`
}

type PipelineItem struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	ID      string `json:"id"`
}

type PipelineRunInput struct {
	ID        string            `json:"id" jsonschema:"Pipeline ID to trigger"`
	Branch    string            `json:"branch,omitempty" jsonschema:"Branch to build"`
	Variables map[string]string `json:"variables,omitempty" jsonschema:"Key-value variables to pass to the build"`
}

type PipelineRunResult struct {
	BuildID string `json:"build_id"`
}

// --- builds ---

func registerBuilds(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_build_progress",
			mcp.WithDescription("Get the progress document of a Codefresh build, including step status and logs"),
			mcp.WithInputSchema[BuildProgressInput](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in BuildProgressInput) (*cfutils.Progress, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}
			progress, err := client.BuildProgress(ctx, in.BuildID)
			if err != nil {
				return nil, fmt.Errorf("getting build progress: %w", err)
			}
			return progress, nil
		}),
	)
}

type BuildProgressInput struct {
	BuildID string `json:"build_id" jsonschema:"Build ID to fetch progress for"`
}

// --- workflows ---

func registerWorkflows(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_workflow_list",
			mcp.WithDescription("List Codefresh workflows, optionally filtered by runtime and project"),
			mcp.WithInputSchema[WorkflowListInput](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in WorkflowListInput) ([]WorkflowItem, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			filterArgs := platmodel.WorkflowsFilterArgs{}
			if in.Runtime != "" {
				filterArgs.Runtime = &in.Runtime
			}
			if in.Project != "" {
				filterArgs.Project = &in.Project
			}

			workflows, err := client.GraphQL().Workflow().List(ctx, filterArgs)
			if err != nil {
				return nil, fmt.Errorf("listing workflows: %w", err)
			}
			items := make([]WorkflowItem, len(workflows))
			for i, w := range workflows {
				items[i] = WorkflowItem{
					Name:    w.Metadata.Name,
					Runtime: w.Metadata.Runtime,
				}
				if w.Metadata.Namespace != nil {
					items[i].Namespace = *w.Metadata.Namespace
				}
				if w.Metadata.UID != nil {
					items[i].UID = *w.Metadata.UID
				}
			}
			return items, nil
		}),
	)

	s.AddTool(
		mcp.NewTool("codefresh_workflow_get",
			mcp.WithDescription("Get a specific Codefresh workflow by UID"),
			mcp.WithInputSchema[WorkflowGetInput](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in WorkflowGetInput) (*WorkflowDetail, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			workflow, err := client.GraphQL().Workflow().Get(ctx, in.UID)
			if err != nil {
				return nil, fmt.Errorf("getting workflow: %w", err)
			}
			if workflow == nil {
				return nil, nil
			}
			detail := &WorkflowDetail{Name: workflow.Metadata.Name}
			if w := workflow.Metadata.Namespace; w != nil {
				detail.Namespace = *w
			}
			if w := workflow.Metadata.UID; w != nil {
				detail.UID = *w
			}
			return detail, nil
		}),
	)
}

type WorkflowListInput struct {
	Runtime string `json:"runtime,omitempty" jsonschema:"Filter by runtime name"`
	Project string `json:"project,omitempty" jsonschema:"Filter by project name"`
}

type WorkflowItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Runtime   string `json:"runtime"`
	UID       string `json:"uid,omitempty"`
}

type WorkflowGetInput struct {
	UID string `json:"uid" jsonschema:"Workflow UID to retrieve"`
}

type WorkflowDetail struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	UID       string `json:"uid,omitempty"`
}

// --- accounts ---

func registerAccounts(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_account_get",
			mcp.WithDescription("Get the active Codefresh account information"),
			mcp.WithInputSchema[struct{}](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) (*AccountInfo, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			user, err := client.GraphQL().User().GetCurrent(ctx)
			if err != nil {
				return nil, fmt.Errorf("getting current user: %w", err)
			}
			account := user.ActiveAccount
			if account == nil {
				return &AccountInfo{}, nil
			}
			info := AccountInfo{ID: account.ID}
			if account.Name != nil {
				info.Name = *account.Name
			}
			return &info, nil
		}),
	)
}

type AccountInfo struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// --- runtimes ---

func registerRuntimes(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_runtime_list",
			mcp.WithDescription("List Codefresh runtimes"),
			mcp.WithInputSchema[struct{}](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) ([]RuntimeItem, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			runtimes, err := client.GraphQL().Runtime().List(ctx)
			if err != nil {
				return nil, fmt.Errorf("listing runtimes: %w", err)
			}
			items := make([]RuntimeItem, len(runtimes))
			for i, r := range runtimes {
				items[i] = RuntimeItem{
					Name:             r.Metadata.Name,
					InstallationType: string(r.InstallationType),
				}
			}
			return items, nil
		}),
	)
}

type RuntimeItem struct {
	Name             string `json:"name"`
	InstallationType string `json:"installation_type,omitempty"`
}

// --- clusters ---

func registerClusters(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_cluster_list",
			mcp.WithDescription("List Codefresh clusters for the active account"),
			mcp.WithInputSchema[struct{}](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, _ struct{}) ([]ClusterItem, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			clusters, err := client.Rest().Cluster().GetAccountClusters()
			if err != nil {
				return nil, fmt.Errorf("listing clusters: %w", err)
			}
			items := make([]ClusterItem, len(clusters))
			for i, c := range clusters {
				items[i] = ClusterItem{
					Name:     c.Cluster.Name,
					Provider: c.Provider,
					Selector: c.Selector,
				}
			}
			return items, nil
		}),
	)
}

type ClusterItem struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Selector string `json:"selector"`
}

// --- git sources ---

func registerGitSources(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("codefresh_git_context_list",
			mcp.WithDescription("List Codefresh git contexts"),
			mcp.WithInputSchema[struct{}](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewStructuredToolHandler(func(_ context.Context, _ mcp.CallToolRequest, _ struct{}) ([]GitContextItem, error) {
			client, err := newClient()
			if err != nil {
				return nil, err
			}

			contexts, err := client.Rest().Context().GetGitContexts()
			if err != nil {
				return nil, fmt.Errorf("listing git contexts: %w", err)
			}
			items := make([]GitContextItem, len(contexts))
			for i, c := range contexts {
				items[i] = GitContextItem{
					Name: c.Metadata.Name,
					Type: c.Spec.Type,
				}
			}
			return items, nil
		}),
	)
}

type GitContextItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

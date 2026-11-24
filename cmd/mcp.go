package main

import (
	"context"
	"encoding/json"
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
			slog.Debug("starting MCP server", "browser", flags.Browser)

			s := server.NewMCPServer("cfutils", "0.1.0",
				server.WithInstructions("Codefresh utilities exposed over MCP. Tools return structured JSON."),
				server.WithToolCapabilities(false),
				server.WithResourceCapabilities(false, false),
				server.WithPromptCapabilities(false),
				server.WithRecovery(),
			)

			registerPipelines(s)
			registerBuilds(s)
			registerWorkflows(s)
			registerAccounts(s)
			registerRuntimes(s)
			registerClusters(s)
			registerGitSources(s)

			if err := server.ServeStdio(s); err != nil {
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
	accessToken, _, err := cfutils.TokensFromBrowser(flags.Browser)
	if err != nil {
		return nil, fmt.Errorf("reading tokens from %s: %w", flags.Browser, err)
	}
	return cfutils.New("", accessToken), nil
}

func textResult(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("serializing result: %w", err)
	}
	return string(b), nil
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
		mcp.NewTypedToolHandler(func(_ context.Context, _ mcp.CallToolRequest, in PipelineListInput) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			filters := map[string]string{"limit": "50"}
			if in.Name != "" {
				filters["id"] = in.Name
			}
			pipelines, err := client.Rest().Pipeline().List(filters)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("listing pipelines: %v", err)), nil
			}

			type item struct {
				Name    string `json:"name"`
				Project string `json:"project"`
				ID      string `json:"id"`
			}
			items := make([]item, len(pipelines))
			for i, p := range pipelines {
				items[i] = item{
					Name:    p.Metadata.Name,
					Project: p.Metadata.Project,
					ID:      p.Metadata.ID,
				}
			}
			result, _ := textResult(items)
			return mcp.NewToolResultText(result), nil
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
		mcp.NewTypedToolHandler(func(_ context.Context, _ mcp.CallToolRequest, in PipelineRunInput) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			buildID, err := client.Rest().Pipeline().Run(in.ID, &rest.RunOptions{
				Branch:    in.Branch,
				Variables: in.Variables,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("running pipeline: %v", err)), nil
			}

			result, _ := textResult(map[string]string{"build_id": buildID})
			return mcp.NewToolResultText(result), nil
		}),
	)
}

type PipelineListInput struct {
	Name string `json:"name" jsonschema:"Filter pipelines by name (loose match)"`
}

type PipelineRunInput struct {
	ID        string            `json:"id" jsonschema:"Pipeline ID to trigger"`
	Branch    string            `json:"branch,omitempty" jsonschema:"Branch to build"`
	Variables map[string]string `json:"variables,omitempty" jsonschema:"Key-value variables to pass to the build"`
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
		mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in BuildProgressInput) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			progress, err := client.BuildProgress(ctx, in.BuildID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("getting build progress: %v", err)), nil
			}
			result, _ := textResult(progress)
			return mcp.NewToolResultText(result), nil
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
		mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in WorkflowListInput) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
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
				return mcp.NewToolResultError(fmt.Sprintf("listing workflows: %v", err)), nil
			}

			type item struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace,omitempty"`
				Runtime   string `json:"runtime"`
				UID       string `json:"uid,omitempty"`
			}
			items := make([]item, len(workflows))
			for i, w := range workflows {
				items[i] = item{
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
			result, _ := textResult(items)
			return mcp.NewToolResultText(result), nil
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
		mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in WorkflowGetInput) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			workflow, err := client.GraphQL().Workflow().Get(ctx, in.UID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("getting workflow: %v", err)), nil
			}
			if workflow == nil {
				return mcp.NewToolResultText("null"), nil
			}

			type detail struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace,omitempty"`
				UID       string `json:"uid,omitempty"`
			}
			d := detail{Name: workflow.Metadata.Name}
			if w := workflow.Metadata.Namespace; w != nil {
				d.Namespace = *w
			}
			if w := workflow.Metadata.UID; w != nil {
				d.UID = *w
			}
			result, _ := textResult(d)
			return mcp.NewToolResultText(result), nil
		}),
	)
}

type WorkflowListInput struct {
	Runtime string `json:"runtime,omitempty" jsonschema:"Filter by runtime name"`
	Project string `json:"project,omitempty" jsonschema:"Filter by project name"`
}

type WorkflowGetInput struct {
	UID string `json:"uid" jsonschema:"Workflow UID to retrieve"`
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
		mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			user, err := client.GraphQL().User().GetCurrent(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("getting current user: %v", err)), nil
			}
			account := user.ActiveAccount
			if account == nil {
				result, _ := textResult(map[string]string{})
				return mcp.NewToolResultText(result), nil
			}

			type info struct {
				Name string `json:"name,omitempty"`
				ID   string `json:"id,omitempty"`
			}
			i := info{ID: account.ID}
			if account.Name != nil {
				i.Name = *account.Name
			}
			result, _ := textResult(i)
			return mcp.NewToolResultText(result), nil
		}),
	)
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
		mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			runtimes, err := client.GraphQL().Runtime().List(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("listing runtimes: %v", err)), nil
			}

			type item struct {
				Name             string `json:"name"`
				InstallationType string `json:"installation_type,omitempty"`
			}
			items := make([]item, len(runtimes))
			for i, r := range runtimes {
				items[i] = item{
					Name:             r.Metadata.Name,
					InstallationType: string(r.InstallationType),
				}
			}
			result, _ := textResult(items)
			return mcp.NewToolResultText(result), nil
		}),
	)
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
		mcp.NewTypedToolHandler(func(_ context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			clusters, err := client.Rest().Cluster().GetAccountClusters()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("listing clusters: %v", err)), nil
			}

			type item struct {
				Name     string `json:"name"`
				Provider string `json:"provider"`
				Selector string `json:"selector"`
			}
			items := make([]item, len(clusters))
			for i, c := range clusters {
				items[i] = item{
					Name:     c.Cluster.Name,
					Provider: c.Provider,
					Selector: c.Selector,
				}
			}
			result, _ := textResult(items)
			return mcp.NewToolResultText(result), nil
		}),
	)
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
		mcp.NewTypedToolHandler(func(_ context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			client, err := newClient()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			contexts, err := client.Rest().Context().GetGitContexts()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("listing git contexts: %v", err)), nil
			}

			type item struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			items := make([]item, len(contexts))
			for i, c := range contexts {
				items[i] = item{
					Name: c.Metadata.Name,
					Type: c.Spec.Type,
				}
			}
			result, _ := textResult(items)
			return mcp.NewToolResultText(result), nil
		}),
	)
}
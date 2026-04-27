package admincmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GizClaw/gizclaw-go/cmd/internal/client"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/spf13/cobra"
)

type resourceClient interface {
	ApplyResource(context.Context, apitypes.Resource) (apitypes.ApplyResult, error)
	DeleteResource(context.Context, apitypes.ResourceKind, string) (apitypes.Resource, error)
	GetResource(context.Context, apitypes.ResourceKind, string) (apitypes.Resource, error)
	Close() error
}

type adminResourceAPI interface {
	ApplyResourceWithResponse(ctx context.Context, body adminservice.ApplyResourceJSONRequestBody, reqEditors ...adminservice.RequestEditorFn) (*adminservice.ApplyResourceResponse, error)
	DeleteResourceWithResponse(ctx context.Context, kind adminservice.ResourceKind, name adminservice.ResourceName, reqEditors ...adminservice.RequestEditorFn) (*adminservice.DeleteResourceResponse, error)
	GetResourceWithResponse(ctx context.Context, kind adminservice.ResourceKind, name adminservice.ResourceName, reqEditors ...adminservice.RequestEditorFn) (*adminservice.GetResourceResponse, error)
}

type resourceClientBridge struct {
	api   adminResourceAPI
	close func() error
}

func (r *resourceClientBridge) ApplyResource(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	resp, err := r.api.ApplyResourceWithResponse(ctx, resource)
	if err != nil {
		return apitypes.ApplyResult{}, err
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return apitypes.ApplyResult{}, resourceResponseError(resp.StatusCode(), resp.Body, resp.JSON400, resp.JSON409, resp.JSON500, resp.JSON501)
}

func (r *resourceClientBridge) DeleteResource(ctx context.Context, kind apitypes.ResourceKind, name string) (apitypes.Resource, error) {
	resp, err := r.api.DeleteResourceWithResponse(ctx, kind, adminservice.ResourceName(name))
	if err != nil {
		return apitypes.Resource{}, err
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return apitypes.Resource{}, resourceResponseError(resp.StatusCode(), resp.Body, resp.JSON400, resp.JSON404, resp.JSON409, resp.JSON500)
}

func (r *resourceClientBridge) GetResource(ctx context.Context, kind apitypes.ResourceKind, name string) (apitypes.Resource, error) {
	resp, err := r.api.GetResourceWithResponse(ctx, kind, adminservice.ResourceName(name))
	if err != nil {
		return apitypes.Resource{}, err
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return apitypes.Resource{}, resourceResponseError(resp.StatusCode(), resp.Body, resp.JSON400, resp.JSON404, resp.JSON500, resp.JSON501)
}

func (r *resourceClientBridge) Close() error {
	if r.close == nil {
		return nil
	}
	return r.close()
}

var openResourceClient = func(ctxName string) (resourceClient, error) {
	c, err := client.ConnectFromContext(ctxName)
	if err != nil {
		return nil, err
	}
	api, err := c.ServerAdminClient()
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	return &resourceClientBridge{
		api:   api,
		close: c.Close,
	}, nil
}

func newApplyCmd(ctxName *string) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "apply -f <file>",
		Short: "Apply an admin resource",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("required flag: --file")
			}
			resource, err := readResourceFile(cmd, file)
			if err != nil {
				return err
			}
			c, err := openResourceClient(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			result, err := c.ApplyResource(context.Background(), resource)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "resource JSON file, or '-' for stdin")
	cmd.Flags().StringVar(ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newShowCmd(ctxName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <kind> <name>",
		Short: "Show a named admin resource",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, name, err := parseNamedResourceArgs(args)
			if err != nil {
				return err
			}
			c, err := openResourceClient(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			resource, err := c.GetResource(context.Background(), kind, name)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(resource)
		},
	}
	cmd.Flags().StringVar(ctxName, "context", "", "context name (default: current)")
	return cmd
}

func newDeleteCmd(ctxName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <kind> <name>",
		Short: "Delete a named admin resource",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, name, err := parseNamedResourceArgs(args)
			if err != nil {
				return err
			}
			c, err := openResourceClient(*ctxName)
			if err != nil {
				return err
			}
			defer c.Close()
			resource, err := c.DeleteResource(context.Background(), kind, name)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(resource)
		},
	}
	cmd.Flags().StringVar(ctxName, "context", "", "context name (default: current)")
	return cmd
}

func readResourceFile(cmd *cobra.Command, path string) (apitypes.Resource, error) {
	var reader io.Reader
	if path == "-" {
		reader = cmd.InOrStdin()
	} else {
		file, err := os.Open(path)
		if err != nil {
			return apitypes.Resource{}, err
		}
		defer file.Close()
		reader = file
	}
	var resource apitypes.Resource
	if err := json.NewDecoder(reader).Decode(&resource); err != nil {
		return apitypes.Resource{}, err
	}
	return resource, nil
}

func parseNamedResourceArgs(args []string) (apitypes.ResourceKind, string, error) {
	kind := apitypes.ResourceKind(args[0])
	if !kind.Valid() {
		return "", "", fmt.Errorf("unknown resource kind %q", args[0])
	}
	if kind == apitypes.ResourceKindResourceList {
		return "", "", fmt.Errorf("resource kind %q cannot be addressed by name", kind)
	}
	name := strings.TrimSpace(args[1])
	if name == "" {
		return "", "", fmt.Errorf("resource name is required")
	}
	return kind, name, nil
}

func resourceResponseError(status int, body []byte, errs ...interface{}) error {
	for _, errResp := range errs {
		switch e := errResp.(type) {
		case *apitypes.ErrorResponse:
			if e != nil {
				return fmt.Errorf("%s: %s", e.Error.Code, e.Error.Message)
			}
		}
	}
	text := strings.TrimSpace(string(body))
	if text != "" {
		return fmt.Errorf("unexpected status %d: %s", status, text)
	}
	if status != 0 {
		return fmt.Errorf("unexpected status %d", status)
	}
	return fmt.Errorf("unexpected empty response")
}

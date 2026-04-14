package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/akatsuki-kk/gmail-collector/internal/auth"
	"github.com/akatsuki-kk/gmail-collector/internal/config"
	"github.com/akatsuki-kk/gmail-collector/internal/gmailclient"
)

type RunOptions struct {
	ConfigPath string
	OutputPath string
}

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx context.Context, opts RunOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	credentials, token, err := auth.EnsureOAuthFiles(auth.PromptIO{
		Stdin:  os.Stdin,
		Stdout: r.Stdout,
		Stderr: r.Stderr,
	})
	if err != nil {
		return err
	}

	service, err := gmailclient.NewService(ctx, credentials, token)
	if err != nil {
		return err
	}

	query := cfg.Search.BuildQuery()
	results, err := gmailclient.Collect(ctx, service, query, cfg.Search.IncludeSpamTrash, cfg.Extract)
	if err != nil {
		return err
	}

	fmt.Fprintf(r.Stderr, "検索結果: %d件\n", len(results))

	var encoded []byte
	if cfg.Output.Pretty {
		encoded, err = json.MarshalIndent(results, "", "  ")
	} else {
		encoded, err = json.Marshal(results)
	}
	if err != nil {
		return err
	}

	if opts.OutputPath == "" {
		_, err = fmt.Fprintln(r.Stdout, string(encoded))
		return err
	}

	return os.WriteFile(opts.OutputPath, append(encoded, '\n'), 0o644)
}

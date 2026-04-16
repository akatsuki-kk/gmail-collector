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

	queries := cfg.Search.BuildQueries()
	listedCounts := make([]int, 0, len(queries))
	queryMessageIDs := make([][]string, 0, len(queries))
	totalListed := 0
	for _, query := range queries {
		ids, err := gmailclient.ListMessageIDs(ctx, service, query, cfg.Search.IncludeSpamTrash)
		if err != nil {
			return err
		}
		listedCounts = append(listedCounts, len(ids))
		queryMessageIDs = append(queryMessageIDs, ids)
		totalListed += len(ids)
	}
	fmt.Fprintf(r.Stderr, "検索対象: %d件\n", totalListed)

	results := make([]gmailclient.Result, 0, totalListed)
	totalProcessed := 0
	for index, query := range queries {
		batch, err := gmailclient.Collect(ctx, service, gmailclient.CollectOptions{
			Query:            query,
			MessageIDs:       queryMessageIDs[index],
			IncludeSpamTrash: cfg.Search.IncludeSpamTrash,
			BodyContains:     cfg.Search.BodyContains,
			ExtractRules:     cfg.Extract,
			OnProgress: func(progress gmailclient.Progress) {
				fmt.Fprintf(r.Stderr, "\r%s", formatProgress(gmailclient.Progress{
					Total:     totalListed,
					Processed: totalProcessed + progress.Processed,
					Matched:   len(results) + progress.Matched,
				}))
				if totalProcessed+progress.Processed == totalListed {
					fmt.Fprintln(r.Stderr)
				}
			},
		})
		if err != nil {
			return err
		}
		results = mergeResults(results, batch)
		totalProcessed += listedCounts[index]
	}

	fmt.Fprintf(r.Stderr, "抽出結果: %d件\n", len(results))

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

func formatProgress(progress gmailclient.Progress) string {
	return fmt.Sprintf("処理中: %d/%d件 (抽出対象: %d件)", progress.Processed, progress.Total, progress.Matched)
}

func mergeResults(current []gmailclient.Result, incoming []gmailclient.Result) []gmailclient.Result {
	seen := make(map[string]struct{}, len(current))
	for _, result := range current {
		seen[result.MessageID] = struct{}{}
	}

	for _, result := range incoming {
		if _, exists := seen[result.MessageID]; exists {
			continue
		}
		current = append(current, result)
		seen[result.MessageID] = struct{}{}
	}

	return current
}

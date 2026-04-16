package gmailclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/akatsuki-kk/gmail-collector/internal/auth"
	"github.com/akatsuki-kk/gmail-collector/internal/extract"
	xhtml "golang.org/x/net/html"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Result struct {
	MessageID string            `json:"message_id"`
	ThreadID  string            `json:"thread_id"`
	Subject   string            `json:"subject"`
	From      string            `json:"from"`
	Date      string            `json:"date"`
	Extracted map[string]string `json:"extracted"`
}

type Progress struct {
	Total     int
	Processed int
	Matched   int
}

type CollectOptions struct {
	Query            string
	MessageIDs       []string
	IncludeSpamTrash bool
	BodyContains     []string
	ExtractRules     map[string]string
	OnListed         func(total int)
	OnProgress       func(Progress)
}

func NewService(ctx context.Context, credentials *auth.StoredCredentials, token *oauth2.Token) (*gmail.Service, error) {
	client := credentials.OAuth2Config().Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}
	return service, nil
}

func Collect(ctx context.Context, service *gmail.Service, opts CollectOptions) ([]Result, error) {
	ids := opts.MessageIDs
	if ids == nil {
		var err error
		ids, err = ListMessageIDs(ctx, service, opts.Query, opts.IncludeSpamTrash)
		if err != nil {
			return nil, err
		}
		if opts.OnListed != nil {
			opts.OnListed(len(ids))
		}
	}

	results := make([]Result, 0, len(ids))
	for index, id := range ids {
		msg, err := service.Users.Messages.Get("me", id).Context(ctx).Format("full").Do()
		if err != nil {
			return nil, fmt.Errorf("fetch message %s: %w", id, err)
		}

		body, err := extractBody(msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("extract body %s: %w", id, err)
		}

		if containsAll(body, opts.BodyContains) {
			extracted, err := extract.Apply(body, opts.ExtractRules)
			if err != nil {
				return nil, err
			}

			results = append(results, Result{
				MessageID: msg.Id,
				ThreadID:  msg.ThreadId,
				Subject:   headerValue(msg.Payload.Headers, "Subject"),
				From:      headerValue(msg.Payload.Headers, "From"),
				Date:      formatDate(headerValue(msg.Payload.Headers, "Date")),
				Extracted: extracted,
			})
		}

		if opts.OnProgress != nil {
			opts.OnProgress(Progress{
				Total:     len(ids),
				Processed: index + 1,
				Matched:   len(results),
			})
		}
	}

	return results, nil
}

func ListMessageIDs(ctx context.Context, service *gmail.Service, query string, includeSpamTrash bool) ([]string, error) {
	call := service.Users.Messages.List("me").Context(ctx).Q(query).IncludeSpamTrash(includeSpamTrash)
	var ids []string

	for {
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("search gmail messages: %w", err)
		}
		for _, msg := range resp.Messages {
			ids = append(ids, msg.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		call.PageToken(resp.NextPageToken)
	}

	return ids, nil
}

func headerValue(headers []*gmail.MessagePartHeader, name string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func formatDate(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := mailDateLayouts(raw)
	if err != nil {
		return raw
	}
	return parsed.Format(time.RFC3339)
}

func mailDateLayouts(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func extractBody(part *gmail.MessagePart) (string, error) {
	if part == nil {
		return "", nil
	}

	if plain := firstBodyByMime(part, "text/plain"); plain != "" {
		return plain, nil
	}

	if htmlBody := firstBodyByMime(part, "text/html"); htmlBody != "" {
		return htmlToText(htmlBody)
	}

	return "", nil
}

func firstBodyByMime(part *gmail.MessagePart, mimeType string) string {
	if strings.EqualFold(part.MimeType, mimeType) {
		return decodeBody(part.Body)
	}

	for _, child := range part.Parts {
		if body := firstBodyByMime(child, mimeType); body != "" {
			return body
		}
	}

	return ""
}

func decodeBody(body *gmail.MessagePartBody) string {
	if body == nil || body.Data == "" {
		return ""
	}

	decoded, err := base64.RawURLEncoding.DecodeString(body.Data)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(body.Data)
	}
	if err != nil {
		return ""
	}
	return string(decoded)
}

func htmlToText(source string) (string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var builder strings.Builder

	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			text := normalizeWhitespace(builder.String())
			return text, nil
		case xhtml.TextToken:
			text := strings.TrimSpace(html.UnescapeString(string(tokenizer.Text())))
			if text == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(text)
		}
	}
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func containsAll(body string, required []string) bool {
	for _, term := range required {
		trimmed := strings.TrimSpace(term)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(body, trimmed) {
			return false
		}
	}
	return true
}

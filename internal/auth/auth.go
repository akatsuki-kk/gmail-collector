package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	appDirName         = "gmail-collector"
	credentialsName    = "credentials.json"
	tokenName          = "token.json"
	defaultRedirectURI = "http://127.0.0.1:8787/callback"
)

type PromptIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type StoredCredentials struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
}

func EnsureOAuthFiles(io PromptIO) (*StoredCredentials, *oauth2.Token, error) {
	dir, err := userConfigDir()
	if err != nil {
		return nil, nil, err
	}

	credentialsPath := filepath.Join(dir, credentialsName)
	tokenPath := filepath.Join(dir, tokenName)

	credentials, err := loadCredentials(credentialsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("load credentials: %w", err)
		}
		credentials, err = promptCredentials(io)
		if err != nil {
			return nil, nil, err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, nil, fmt.Errorf("create config dir: %w", err)
		}
		if err := writeJSON(credentialsPath, credentials, 0o600); err != nil {
			return nil, nil, fmt.Errorf("save credentials: %w", err)
		}
		fmt.Fprintf(io.Stderr, "認証情報を保存しました: %s\n", credentialsPath)
	}

	token, err := loadToken(tokenPath)
	if err == nil {
		return credentials, token, nil
	}
	if !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("load token: %w", err)
	}

	token, err = promptOAuthToken(context.Background(), io, credentials)
	if err != nil {
		return nil, nil, err
	}
	if err := writeJSON(tokenPath, token, 0o600); err != nil {
		return nil, nil, fmt.Errorf("save token: %w", err)
	}
	fmt.Fprintf(io.Stderr, "トークンを保存しました: %s\n", tokenPath)

	return credentials, token, nil
}

func promptCredentials(io PromptIO) (*StoredCredentials, error) {
	reader := bufio.NewReader(io.Stdin)

	fmt.Fprintln(io.Stdout, "OAuth 認証情報を入力してください。")

	clientID, err := promptLine(reader, io.Stdout, "Google OAuth Client ID: ")
	if err != nil {
		return nil, err
	}
	clientSecret, err := promptLine(reader, io.Stdout, "Google OAuth Client Secret: ")
	if err != nil {
		return nil, err
	}
	redirectURI, err := promptLine(reader, io.Stdout, fmt.Sprintf("Redirect URI [%s]: ", defaultRedirectURI))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = defaultRedirectURI
	}

	return &StoredCredentials{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		RedirectURI:  strings.TrimSpace(redirectURI),
		Scopes:       []string{"https://www.googleapis.com/auth/gmail.readonly"},
	}, nil
}

func promptOAuthToken(ctx context.Context, io PromptIO, credentials *StoredCredentials) (*oauth2.Token, error) {
	reader := bufio.NewReader(io.Stdin)
	cfg := credentials.OAuth2Config()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server := &http.Server{}
	listener, err := net.Listen("tcp", "127.0.0.1:8787")
	if err == nil {
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				errCh <- fmt.Errorf("callback did not include authorization code")
				return
			}
			fmt.Fprintln(w, "認証が完了しました。CLI に戻ってください。")
			codeCh <- code
		})
		go func() {
			_ = server.Serve(listener)
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		}()
	} else {
		fmt.Fprintf(io.Stderr, "ローカルコールバックを開始できませんでした。手動でコード入力に切り替えます: %v\n", err)
	}

	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Fprintln(io.Stdout, "以下のURLをブラウザで開いて認証してください。")
	fmt.Fprintln(io.Stdout, authURL)
	fmt.Fprintln(io.Stdout, "ブラウザから戻らない場合は、認可コードを貼り付けて Enter を押してください。")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(2 * time.Second):
		manualCode, err := promptLine(reader, io.Stdout, "Authorization code (空なら待機): ")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(manualCode) != "" {
			code = manualCode
		} else {
			select {
			case code = <-codeCh:
			case err := <-errCh:
				return nil, err
			case <-time.After(5 * time.Minute):
				return nil, fmt.Errorf("timed out waiting for authorization")
			}
		}
	}

	token, err := cfg.Exchange(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}

	return token, nil
}

func (c StoredCredentials) OAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURI,
		Scopes:       c.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}

func userConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func loadCredentials(path string) (*StoredCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var credentials StoredCredentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &credentials, nil
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return &token, nil
}

func writeJSON(path string, value any, perm os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, perm)
}

func promptLine(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprint(out, label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

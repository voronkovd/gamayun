package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/voronkovd/gamayun/internal/version"
)

type Release struct {
	Tag    string
	Assets map[string]string
	Sums   map[string]string
}

type Client struct {
	HTTP    *http.Client
	API     string
	Repo    string
	Current string
	Dest    string
	GOARCH  string
	Restart func() error
}

func DefaultClient(repo, dest string) *Client {
	if dest == "" {
		dest, _ = os.Executable()
	}
	return &Client{
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		Repo:    repo,
		Current: version.Version,
		Dest:    dest,
		GOARCH:  runtime.GOARCH,
		Restart: restartSystemd,
	}
}

func (c *Client) Run(ctx context.Context) error {
	if c.Repo == "" {
		return fmt.Errorf("github repo unknown: pass --repo owner/name, set update.github_repo, or rebuild from CI")
	}
	if InstalledViaApt(c.Dest) {
		return fmt.Errorf("binary is owned by apt; update with: sudo apt update && sudo apt install --only-upgrade gamayun")
	}
	rel, err := c.Latest(ctx)
	if err != nil {
		return err
	}
	if !NeedsUpdate(c.Current, rel.Tag) {
		fmt.Printf("already up to date (%s)\n", c.Current)
		return nil
	}
	asset := "gamayun-linux-" + c.arch()
	url, ok := rel.Assets[asset]
	if !ok {
		return fmt.Errorf("release %s has no asset %s", rel.Tag, asset)
	}
	fmt.Printf("updating %s → %s\n", c.Current, rel.Tag)
	tmp, err := c.download(ctx, url)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if want, ok := rel.Sums[asset]; ok {
		if err := verifySHA256(tmp, want); err != nil {
			return err
		}
	}
	if err := replaceFile(c.Dest, tmp); err != nil {
		return err
	}
	fmt.Printf("installed %s to %s\n", rel.Tag, c.Dest)
	if c.Restart != nil {
		if err := c.Restart(); err != nil {
			fmt.Fprintf(os.Stderr, "restart skipped: %v\n", err)
		}
	}
	return nil
}

func (c *Client) Latest(ctx context.Context) (Release, error) {
	root := "https://api.github.com"
	if c.API != "" {
		root = strings.TrimRight(c.API, "/")
	}
	api := root + "/repos/" + c.Repo + "/releases/latest"
	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := c.getJSON(ctx, api, &payload); err != nil {
		return Release{}, err
	}
	if payload.TagName == "" {
		return Release{}, fmt.Errorf("no releases in %s", c.Repo)
	}
	rel := Release{Tag: payload.TagName, Assets: map[string]string{}}
	for _, a := range payload.Assets {
		rel.Assets[a.Name] = a.URL
	}
	if sumsURL, ok := rel.Assets["SHA256SUMS"]; ok {
		sums, err := c.getSums(ctx, sumsURL)
		if err != nil {
			return rel, fmt.Errorf("SHA256SUMS: %w", err)
		}
		rel.Sums = sums
	}
	return rel, nil
}

func (c *Client) getJSON(ctx context.Context, url string, dest any) error {
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()
	return json.NewDecoder(body).Decode(dest)
}

func (c *Client) getSums(ctx context.Context, url string) (map[string]string, error) {
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return nil, err
	}
	return ParseSHA256SUMS(string(data)), nil
}

func (c *Client) get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gamayun/"+c.Current)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

func (c *Client) download(ctx context.Context, url string) (string, error) {
	body, err := c.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer body.Close()
	f, err := os.CreateTemp("", "gamayun-update-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) arch() string {
	if c.GOARCH != "" {
		return c.GOARCH
	}
	return runtime.GOARCH
}

func NeedsUpdate(current, latest string) bool {
	c := normalizeVer(current)
	l := normalizeVer(latest)
	if l == "" {
		return false
	}
	if c == "" || c == "dev" || strings.Contains(c, "dirty") {
		return true
	}
	return c != l
}

func normalizeVer(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

func ParseSHA256SUMS(s string) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		name := filepath.Base(fields[1])
		out[name] = fields[0]
	}
	return out
}

func verifySHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}
	return nil
}

func replaceFile(dest, src string) error {
	if dest == "" {
		return fmt.Errorf("destination binary unknown")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func InstalledViaApt(path string) bool {
	if path == "" {
		return false
	}
	cmd := exec.Command("dpkg-query", "-S", path)
	return cmd.Run() == nil
}

func restartSystemd() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return err
	}
	cmd := exec.Command("systemctl", "restart", "gamayun")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart: %v: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Println("service gamayun restarted")
	return nil
}

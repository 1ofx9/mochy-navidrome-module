package subsonic

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiVersion = "1.16.1"
	clientName = "mochy-navidrome"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
)

type Client struct {
	baseURL      string
	username     string
	password     string
	httpClient   *http.Client
	streamClient *http.Client
}

func New(baseURL, username, password string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		username:     username,
		password:     password,
		httpClient:   httpClient,
		streamClient: &http.Client{Timeout: 0},
	}
}

// AuthParams constructs the query parameters required for Subsonic MD5 token authentication.
func (c *Client) AuthParams() url.Values {
	salt := randomSalt()
	sum := md5.Sum([]byte(c.password + salt))
	q := url.Values{}
	q.Set("u", c.username)
	q.Set("t", hex.EncodeToString(sum[:]))
	q.Set("s", salt)
	q.Set("v", apiVersion)
	q.Set("c", clientName)
	q.Set("f", "json")
	return q
}

func randomSalt() string {
	raw := make([]byte, 8)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

type responseEnvelope struct {
	Response json.RawMessage `json:"subsonic-response"`
}

type statusHeader struct {
	Status  string    `json:"status"`
	Version string    `json:"version"`
	Error   *APIError `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params url.Values, into any) error {
	q := c.AuthParams()
	for k, v := range params {
		q[k] = v
	}

	reqURL := c.baseURL + "/rest/" + method + ".view?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024))
	if err != nil {
		return err
	}

	var env responseEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("navidrome: invalid json envelope: %w", err)
	}

	var header statusHeader
	if err := json.Unmarshal(env.Response, &header); err != nil {
		return fmt.Errorf("navidrome: invalid response body: %w", err)
	}

	if header.Status != "" && header.Status != "ok" {
		if header.Error != nil {
			switch header.Error.Code {
			case 40, 41, 10, 50:
				return fmt.Errorf("%w: %s", ErrUnauthorized, header.Error.Message)
			case 70:
				return fmt.Errorf("%w: %s", ErrNotFound, header.Error.Message)
			default:
				return fmt.Errorf("navidrome error %d: %s", header.Error.Code, header.Error.Message)
			}
		}
		return fmt.Errorf("navidrome status: %s", header.Status)
	}

	if into == nil {
		return nil
	}
	return json.Unmarshal(env.Response, into)
}

func (c *Client) Ping(ctx context.Context) (string, error) {
	var answer struct {
		Version string `json:"version"`
	}
	if err := c.call(ctx, "ping", nil, &answer); err != nil {
		return "", err
	}
	return answer.Version, nil
}

func (c *Client) Search(ctx context.Context, query string, count int) ([]Song, error) {
	if count <= 0 {
		count = 10
	}
	params := url.Values{
		"query":       {query},
		"songCount":   {strconv.Itoa(count)},
		"artistCount": {"0"},
		"albumCount":  {"0"},
	}

	var answer struct {
		SearchResult3 struct {
			Song SongList `json:"song"`
		} `json:"searchResult3"`
	}

	if err := c.call(ctx, "search3", params, &answer); err != nil {
		return nil, err
	}
	return answer.SearchResult3.Song, nil
}

func (c *Client) GetSong(ctx context.Context, id string) (*Song, error) {
	var answer struct {
		Song Song `json:"song"`
	}
	if err := c.call(ctx, "getSong", url.Values{"id": {id}}, &answer); err != nil {
		return nil, err
	}
	if answer.Song.ID == "" {
		return nil, ErrNotFound
	}
	return &answer.Song, nil
}

func (c *Client) GetLyrics(ctx context.Context, id, artist, title string) (string, string, error) {
	if id != "" {
		var listResp LyricsListResponse
		if err := c.call(ctx, "getLyricsBySongId", url.Values{"id": {id}}, &listResp); err == nil {
			for _, item := range listResp.LyricsList.StructuredLyrics {
				if item.Synced && len(item.Lines) > 0 {
					var b strings.Builder
					for i, line := range item.Lines {
						if i > 0 {
							b.WriteByte('\n')
						}
						b.WriteString(formatLRC(line.Start, line.Value))
					}
					return b.String(), "", nil
				}
			}
		}
	}

	if artist != "" && title != "" {
		var stdResp StandardLyricsResponse
		params := url.Values{
			"artist": {artist},
			"title":  {title},
		}
		if err := c.call(ctx, "getLyrics", params, &stdResp); err == nil && stdResp.Lyrics.Content != "" {
			return "", stdResp.Lyrics.Content, nil
		}
	}

	return "", "", ErrNotFound
}

func formatLRC(startMs int, text string) string {
	if startMs < 0 {
		startMs = 0
	}
	mins := startMs / 60000
	secs := (startMs % 60000) / 1000
	hundredths := (startMs % 1000) / 10
	return fmt.Sprintf("[%02d:%02d.%02d] %s", mins, secs, hundredths, text)
}

func (c *Client) StreamURL(id string, maxBitRate int) string {
	q := c.AuthParams()
	q.Set("id", id)
	if maxBitRate > 0 {
		q.Set("maxBitRate", strconv.Itoa(maxBitRate))
	} else {
		q.Set("format", "raw")
	}
	return c.baseURL + "/rest/stream.view?" + q.Encode()
}

func (c *Client) ArtURL(id string, size int) string {
	if size <= 0 {
		size = 600
	}
	q := c.AuthParams()
	q.Set("id", id)
	q.Set("size", strconv.Itoa(size))
	return c.baseURL + "/rest/getCoverArt.view?" + q.Encode()
}

func (c *Client) OpenStream(ctx context.Context, id string, maxBitRate int, headers http.Header) (*http.Response, error) {
	streamURL := c.StreamURL(id, maxBitRate)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	if headers != nil {
		for _, name := range []string{"Range", "If-Range"} {
			if v := headers.Get(name); v != "" {
				req.Header.Set(name, v)
			}
		}
	}
	return c.streamClient.Do(req)
}

func (c *Client) FetchArt(ctx context.Context, id string, size int) ([]byte, string, string, string, error) {
	artURL := c.ArtURL(id, size)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artURL, nil)
	if err != nil {
		return nil, "", "", "", err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", "", "", err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, "", "", "", ErrNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", "", "", fmt.Errorf("navidrome cover art returned status %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", "", "", err
	}

	ct := res.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return body, ct, res.Header.Get("ETag"), res.Header.Get("Last-Modified"), nil
}

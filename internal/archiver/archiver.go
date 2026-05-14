package archiver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Result struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        []byte
	FetchedAt   time.Time
}

type Archiver struct {
	client      *http.Client
	maxBodySize int64
}

func NewArchiver(timeout time.Duration, maxBodySize int64) *Archiver {
	return &Archiver{
		client: &http.Client{
			Timeout: timeout,
		},
		maxBodySize: maxBodySize,
	}
}

func (a *Archiver) Fetch(ctx context.Context, url string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make a request %w", err)
	}

	req.Header.Set("User-Agent", "link-archiver/0.1 (+https://github.com/Hoaqim/link-archiver)")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("response error %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected statu %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, a.maxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(body)) > a.maxBodySize {
		return nil, fmt.Errorf("body too big")
	}

	return &Result{
		URL:         url,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
		FetchedAt:   time.Now().UTC(),
	}, nil
}

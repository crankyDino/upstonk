// Package ingest collects JSE ETF data into the store: daily prices
// (Yahoo), the JSE listing (imported from a file the user downloads) and
// issuer factsheets.
//
// UPSTONK is for the owner's personal use only, and collects the way the owner
// would by hand (see DOCS/plans/jse-data-store.md): one request at a time,
// seconds apart, with a browser User-Agent. It never defeats bot
// challenges; where a site shows one, use the manual-import path.
package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Fetcher retrieves a URL's body. It's the only network boundary in this
// package, so tests replace it with canned responses.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

// PoliteFetcher fetches sequentially with a minimum gap between requests.
type PoliteFetcher struct {
	Client    *http.Client
	UserAgent string
	Gap       time.Duration

	mu   sync.Mutex
	last time.Time
}

// NewPoliteFetcher returns a fetcher that waits gap between requests.
func NewPoliteFetcher(gap time.Duration) *PoliteFetcher {
	return &PoliteFetcher{
		Client:    &http.Client{Timeout: 60 * time.Second},
		UserAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0",
		Gap:       gap,
	}
}

func (f *PoliteFetcher) Get(ctx context.Context, url string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if wait := f.Gap - time.Since(f.last); wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer func() { f.last = time.Now() }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.UserAgent)
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 50<<20))
}

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	attempts     = 3
	maxBodyBytes = 32 << 20
)

type Client struct {
	HTTP    *http.Client
	Backoff time.Duration
}

func New() Client {
	return Client{HTTP: &http.Client{Timeout: 60 * time.Second}, Backoff: 2 * time.Second}
}

type statusError struct {
	code int
}

func (e statusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d", e.code)
}

func (c Client) JSON(ctx context.Context, url string, v any) error {
	var errs []error
	for attempt := range attempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("fetch %s: %w", url, errors.Join(append(errs, ctx.Err())...))
			case <-time.After(c.Backoff * time.Duration(attempt)):
			}
		}
		err := c.once(ctx, url, v)
		if err == nil {
			return nil
		}
		errs = append(errs, fmt.Errorf("attempt %d: %w", attempt+1, err))
		if !retryable(err) {
			break
		}
	}
	return fmt.Errorf("fetch %s: %w", url, errors.Join(errs...))
}

func (c Client) once(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError{code: resp.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(v); err != nil {
		return permanent{fmt.Errorf("decode JSON: %w", err)}
	}
	return nil
}

type permanent struct {
	error
}

func (p permanent) Unwrap() error {
	return p.error
}

func retryable(err error) bool {
	var p permanent
	if errors.As(err, &p) {
		return false
	}
	var s statusError
	if errors.As(err, &s) {
		return s.code == http.StatusTooManyRequests || s.code >= 500
	}
	return true
}

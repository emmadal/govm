package pkg

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultIdleTimeout is how long an attempt may go without receiving a
	// single byte before it is considered stalled.
	DefaultIdleTimeout = 60 * time.Second
	// DefaultMaxAttempts is the total number of download attempts.
	DefaultMaxAttempts = 3
	// DefaultRetryDelay is the pause before the second attempt; it grows
	// linearly with each further attempt.
	DefaultRetryDelay = time.Second
)

// ErrIdleTimeout reports that an attempt received no data for the whole idle
// timeout.
var ErrIdleTimeout = errors.New("timeout: no data received")

// HTTPStatusError reports a response whose status is not 200 OK.
type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// localError marks a failure of the local filesystem. Retrying the download
// cannot fix it.
type localError struct{ err error }

func (e *localError) Error() string { return e.err.Error() }
func (e *localError) Unwrap() error { return e.err }

// Downloader fetches a URL into a file. Transient failures are retried, and a
// failed download never leaves a partial file behind. The zero value is ready
// to use.
type Downloader struct {
	// Client performs the requests. It must not set a global Timeout, which
	// would cut long downloads short. Defaults to a client without one.
	Client *http.Client
	// IdleTimeout defaults to DefaultIdleTimeout.
	IdleTimeout time.Duration
	// MaxAttempts defaults to DefaultMaxAttempts.
	MaxAttempts int
	// RetryDelay defaults to DefaultRetryDelay.
	RetryDelay time.Duration
	// Mode is the permission of the downloaded file. Defaults to 0644.
	Mode os.FileMode
	// Progress, if set, is called once the response headers of an attempt
	// arrive. total is -1 when the size is unknown. Every received byte is
	// also written to the returned writer, unless it is nil.
	Progress func(attempt int, total int64) io.Writer
	// OnRetry, if set, is called before attempt number `attempt` starts,
	// with the error that failed the previous one.
	OnRetry func(attempt, maxAttempts int, err error)
}

// Download fetches url into dest. The data is written to a temporary file next
// to dest and renamed only once it is complete, so dest is either absent,
// untouched, or whole.
func (d *Downloader) Download(ctx context.Context, url, dest string) error {
	maxAttempts := cmp.Or(d.MaxAttempts, DefaultMaxAttempts)
	delay := cmp.Or(d.RetryDelay, DefaultRetryDelay)

	var err error
	for attempt := 1; ; attempt++ {
		err = d.attempt(ctx, url, dest, attempt)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("download cancelled: %w", ctx.Err())
		}
		if !retryable(err) {
			return err
		}
		if attempt >= maxAttempts {
			return fmt.Errorf("download failed after %d attempts: %w", maxAttempts, err)
		}
		if d.OnRetry != nil {
			d.OnRetry(attempt+1, maxAttempts, err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("download cancelled: %w", ctx.Err())
		case <-time.After(delay * time.Duration(attempt)):
		}
	}
}

// retryable reports whether another attempt may succeed.
func retryable(err error) bool {
	if _, ok := errors.AsType[*localError](err); ok {
		return false
	}
	if statusErr, ok := errors.AsType[*HTTPStatusError](err); ok {
		return statusErr.StatusCode >= 500 || statusErr.StatusCode == http.StatusTooManyRequests
	}
	// Timeouts, connection resets and truncated bodies.
	return true
}

func (d *Downloader) attempt(ctx context.Context, url, dest string, attempt int) (err error) {
	idle := cmp.Or(d.IdleTimeout, DefaultIdleTimeout)

	// The watchdog cancels the attempt when no byte arrives for `idle`. It is
	// armed before the request so that waiting for headers counts too.
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	watchdog := time.AfterFunc(idle, func() { cancel(ErrIdleTimeout) })
	defer watchdog.Stop()

	// Report the stall rather than the "context canceled" it surfaces as.
	defer func() {
		if err != nil && errors.Is(context.Cause(ctx), ErrIdleTimeout) {
			err = fmt.Errorf("%w for %s", ErrIdleTimeout, idle)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &localError{err}
	}
	resp, err := cmp.Or(d.Client, &http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &HTTPStatusError{StatusCode: resp.StatusCode}
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".*.part")
	if err != nil {
		return &localError{fmt.Errorf("failed to create temporary file: %w", err)}
	}
	closed := false
	defer func() {
		if err != nil {
			// Close before removing: Windows cannot delete an open file.
			if !closed {
				_ = tmp.Close()
			}
			_ = os.Remove(tmp.Name())
		}
	}()

	var out io.Writer = tmp
	if d.Progress != nil {
		if progress := d.Progress(attempt, resp.ContentLength); progress != nil {
			out = io.MultiWriter(tmp, progress)
		}
	}

	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			watchdog.Reset(idle)
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return &localError{fmt.Errorf("failed to write %s: %w", tmp.Name(), writeErr)}
			}
			written += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	watchdog.Stop()

	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return fmt.Errorf("incomplete download: received %d of %d bytes: %w", written, resp.ContentLength, io.ErrUnexpectedEOF)
	}

	if err = tmp.Chmod(cmp.Or(d.Mode, 0o644)); err != nil {
		return &localError{err}
	}
	closed = true
	if err = tmp.Close(); err != nil {
		return &localError{fmt.Errorf("failed to close %s: %w", tmp.Name(), err)}
	}
	if err = os.Rename(tmp.Name(), dest); err != nil {
		return &localError{err}
	}
	return nil
}

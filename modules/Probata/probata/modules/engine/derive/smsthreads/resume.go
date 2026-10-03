// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// A source object is read in one long GET while the decoder stages and
// uploads its media, so the connection can be held open for many minutes. Live
// 2026-10-02: the 2.8 GB sms-002-031.xml (mostly MMS) failed three attempts
// with a bare "unexpected EOF" about six minutes in, although the file is
// well-formed XML end to end (3,291 records, matching its declared count): the
// provider closed the stream before Content-Length. resumingBody picks the
// stream up where it broke with a ranged GET pinned to the object's ETag, so a
// dropped connection costs a reconnect instead of the whole derivation.

package smsthreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"time"
)

// rangedGet opens the object from byte offset from (0 = the whole object).
// etag, when set, pins the request to that version of the object (If-Match).
// It returns the body, the object's ETag and the total object size (-1 when
// the provider did not say).
type rangedGet func(ctx context.Context, from int64, etag string) (body io.ReadCloser, objectETag string, total int64, err error)

const (
	// maxConsecutiveResumes bounds reconnects that make no progress in between.
	maxConsecutiveResumes = 6
	// resumeProgressReset is how much must be read after a resume before the
	// consecutive-resume count starts again.
	resumeProgressReset = 1 << 20
)

// resumingBody is an io.ReadCloser over one object that reconnects with a
// ranged GET when the stream breaks mid-object.
type resumingBody struct {
	ctx     context.Context
	get     rangedGet
	body    io.ReadCloser
	etag    string
	total   int64
	offset  int64
	sinceOK int64
	resumes int
	// Resumes counts every reconnect, for the caller's receipt.
	Resumes int
	backoff func(attempt int) time.Duration
	sleep   func(ctx context.Context, d time.Duration) error
}

func openResuming(ctx context.Context, get rangedGet) (*resumingBody, error) {
	body, etag, total, err := get(ctx, 0, "")
	if err != nil {
		return nil, err
	}
	return &resumingBody{ctx: ctx, get: get, body: body, etag: etag, total: total,
		backoff: func(attempt int) time.Duration { return time.Duration(attempt*attempt) * time.Second },
		sleep:   sleepContext}, nil
}

func (r *resumingBody) Read(p []byte) (int, error) {
	for {
		if r.body == nil {
			if err := r.reconnect(); err != nil {
				return 0, err
			}
		}
		n, err := r.body.Read(p)
		r.offset += int64(n)
		r.sinceOK += int64(n)
		if r.sinceOK >= resumeProgressReset {
			r.resumes = 0
		}
		switch {
		case err == nil:
			return n, nil
		case errors.Is(err, io.EOF):
			if r.total < 0 || r.offset >= r.total {
				return n, io.EOF
			}
			// The stream ended short of the object's size: a dropped connection.
			err = io.ErrUnexpectedEOF
		}
		if !r.resumable(err) {
			return n, err
		}
		_ = r.body.Close()
		r.body = nil
		if r.resumes >= maxConsecutiveResumes {
			return n, fmt.Errorf("source stream broke at byte %d and %d reconnects made no progress: %w", r.offset, r.resumes, err)
		}
		if n > 0 {
			return n, nil
		}
	}
}

func (r *resumingBody) reconnect() error {
	r.resumes++
	r.Resumes++
	if err := r.sleep(r.ctx, r.backoff(r.resumes)); err != nil {
		return err
	}
	body, etag, _, err := r.get(r.ctx, r.offset, r.etag)
	if err != nil {
		return fmt.Errorf("resume the source stream at byte %d: %w", r.offset, err)
	}
	if r.etag != "" && etag != "" && etag != r.etag {
		_ = body.Close()
		return fmt.Errorf("resume the source stream at byte %d: the object changed (ETag %s, was %s)", r.offset, etag, r.etag)
	}
	r.body, r.sinceOK = body, 0
	return nil
}

// resumable reports whether err is a broken connection rather than a refusal,
// a cancellation or the end of the object.
func (r *resumingBody) resumable(err error) bool {
	if err == nil || r.ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "connection reset") || strings.Contains(message, "unexpected EOF") ||
		strings.Contains(message, "broken pipe") || strings.Contains(message, "stream error")
}

func (r *resumingBody) Close() error {
	if r.body == nil {
		return nil
	}
	err := r.body.Close()
	r.body = nil
	return err
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

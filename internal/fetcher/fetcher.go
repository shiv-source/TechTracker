package fetcher

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/utils"
)

const (
	defaultConcurrency   = 5
	defaultRatePerMinute = 30
	defaultTimeout       = 15 * time.Second
	maxRetries           = 3
)

// RateLimiter implements a simple token-bucket for API rate control.
type RateLimiter struct {
	ticker   *time.Ticker
	tokens   chan struct{}
	done     chan struct{}
}

// NewRateLimiter creates a rate limiter allowing ratePerMinute requests per minute.
func NewRateLimiter(ratePerMinute int) *RateLimiter {
	if ratePerMinute <= 0 {
		ratePerMinute = defaultRatePerMinute
	}
	interval := time.Minute / time.Duration(ratePerMinute)
	rl := &RateLimiter{
		ticker: time.NewTicker(interval),
		tokens: make(chan struct{}, ratePerMinute),
		done:   make(chan struct{}),
	}
	// Pre-fill tokens.
	for i := 0; i < ratePerMinute; i++ {
		rl.tokens <- struct{}{}
	}
	// Refill loop.
	go func() {
		for {
			select {
			case <-rl.ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Bucket full, skip.
				}
			case <-rl.done:
				return
			}
		}
	}()
	return rl
}

// Wait blocks until a token is available or the context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop shuts down the rate limiter.
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.done)
}

// FetchResult holds the result of fetching a single repository.
type FetchResult struct {
	Repo  models.Repository
	Error error
	URL   string
}

// FetchRepository fetches a single GitHub repository with retries and backoff.
func FetchRepository(apiURL, token string, limiter *RateLimiter) (*models.Repository, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			jitter := time.Duration(rand.Int63n(int64(500 * time.Millisecond)))
			time.Sleep(backoff + jitter)
		}

		if limiter != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := limiter.Wait(ctx); err != nil {
				cancel()
				return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
			}
			cancel()
		}

		result, err := makeRequest(apiURL, token)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Only retry on 429 (rate limit) and 5xx errors.
		if !isRetryable(err) {
			break
		}
	}
	return nil, lastErr
}

func makeRequest(apiURL, token string) (*models.Repository, error) {
	return utils.MakeAuthenticatedGETRequest[models.Repository](apiURL, token)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status: 429") ||
		strings.Contains(msg, "status: 403") ||
		strings.Contains(msg, "status: 5")
}

// FetchAll fetches all repository URLs with controlled concurrency and rate limiting.
// Returns partial results even if some requests fail.
func FetchAll(urls []string, token string, concurrency int, verbose bool) ([]models.Repository, []error) {
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	limiter := NewRateLimiter(defaultRatePerMinute)
	defer limiter.Stop()

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrency)
		results = make(chan FetchResult, len(urls))
	)

	for _, url := range urls {
		wg.Add(1)
		go func(apiURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			repo, err := FetchRepository(apiURL, token, limiter)
			if err != nil {
				if verbose {
					fmt.Printf("Error fetching %s: %v\n", apiURL, err)
				}
				results <- FetchResult{
					URL:   apiURL,
					Error: err,
				}
			} else {
				results <- FetchResult{
					URL:  apiURL,
					Repo: *repo,
				}
			}
		}(url)
	}

	wg.Wait()
	close(results)

	var repositories []models.Repository
	var errors []error
	for res := range results {
		if res.Error != nil {
			errors = append(errors, fmt.Errorf("%s: %w", res.URL, res.Error))
		} else {
			repositories = append(repositories, res.Repo)
		}
	}

	return repositories, errors
}

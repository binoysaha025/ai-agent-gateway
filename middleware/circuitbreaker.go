package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CircuitState int 

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu sync.Mutex
	state CircuitState
	failures int
	maxFailures int
	timeout time.Duration
	lastFailTime time.Time
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state: StateClosed,
		maxFailures: maxFailures,
		timeout: timeout,
	}
}

func (cb *CircuitBreaker) recordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailTime = time.Now()

    if cb.failures >= cb.maxFailures {
        cb.state = StateOpen
    }
}

func (cb *CircuitBreaker) recordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.state = StateClosed
}

func (cb *CircuitBreaker) canRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func CircuitBreakerMiddleware(cb *CircuitBreaker) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !cb.canRequest() {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "error":       "service temporarily unavailable",
                "reason":      "circuit breaker open",
                "retry_after": cb.timeout.String(),
            })
            c.Abort()
            return
        }

        c.Next()

        if c.Writer.Status() >= 500 {
            cb.recordFailure()
        } else {
            cb.recordSuccess()
        }
    }
}
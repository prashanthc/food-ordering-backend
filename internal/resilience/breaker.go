package resilience

import (
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
)

var (
	PostgresCB *gobreaker.CircuitBreaker[interface{}]
	RedisCB    *gobreaker.CircuitBreaker[interface{}]
)

func Init() {
	PostgresCB = gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "postgres",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state change", "name", name, "from", from.String(), "to", to.String())
		},
	})

	RedisCB = gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "redis",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state change", "name", name, "from", from.String(), "to", to.String())
		},
	})
}

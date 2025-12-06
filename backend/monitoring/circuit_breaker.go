package monitoring

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// StateClosed - circuit breaker is closed, requests pass through
	StateClosed CircuitBreakerState = iota
	// StateOpen - circuit breaker is open, requests fail fast
	StateOpen
	// StateHalfOpen - circuit breaker is half-open, testing if service recovered
	StateHalfOpen
)

// String returns string representation of circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the circuit breaker is half-open
	MaxRequests uint32

	// Interval is the cyclic period of the closed state
	// for the circuit breaker to clear the internal counts
	Interval time.Duration

	// Timeout is the period of the open state,
	// after which the state of the circuit breaker becomes half-open
	Timeout time.Duration

	// ReadyToTrip is called with a copy of Counts whenever a request fails
	// in the closed state. If ReadyToTrip returns true, the circuit breaker will be placed into the open state
	ReadyToTrip func(counts Counts) bool

	// OnStateChange is called whenever the state of the circuit breaker changes
	OnStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState)

	// IsSuccessful is called with the error returned from a request
	// If IsSuccessful returns true, the request is considered successful
	IsSuccessful func(err error) bool
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// OnRequest increments the number of requests
func (c *Counts) OnRequest() {
	c.Requests++
}

// OnSuccess increments the number of successes
func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// OnFailure increments the number of failures
func (c *Counts) OnFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// Clear resets the counts
func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// CircuitBreaker represents a circuit breaker
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState)

	mutex      sync.Mutex
	state      CircuitBreakerState
	generation uint64
	counts     Counts
	expiry     time.Time

	// Metrics integration
	metrics *PrometheusMetrics
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig, metrics *PrometheusMetrics) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		isSuccessful:  config.IsSuccessful,
		onStateChange: config.OnStateChange,
		state:         StateClosed,
		expiry:        time.Now().Add(config.Interval),
		metrics:       metrics,
	}

	// Set default functions if not provided
	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	}

	if cb.isSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}

	if cb.interval == 0 {
		cb.interval = 60 * time.Second
	}

	if cb.timeout == 0 {
		cb.timeout = 60 * time.Second
	}

	return cb
}

// Execute executes the given request if the circuit breaker accepts it
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, cb.isSuccessful(err))

	return result, err
}

// ExecuteWithFallback executes the request with a fallback function
func (cb *CircuitBreaker) ExecuteWithFallback(req func() (interface{}, error), fallback func(error) (interface{}, error)) (interface{}, error) {
	result, err := cb.Execute(req)
	if err != nil {
		if cb.State() == StateOpen {
			// Circuit breaker is open, use fallback
			return fallback(err)
		}
		return result, err
	}
	return result, nil
}

// beforeRequest is called before a request
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		// Record metrics
		if cb.metrics != nil {
			cb.metrics.RecordCircuitBreakerMetrics(cb.name, int(state), false)
		}
		return generation, errors.New("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		// Record metrics
		if cb.metrics != nil {
			cb.metrics.RecordCircuitBreakerMetrics(cb.name, int(state), false)
		}
		return generation, errors.New("too many requests")
	}

	cb.counts.OnRequest()
	return generation, nil
}

// afterRequest is called after a request
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}

	// Record metrics
	if cb.metrics != nil {
		cb.metrics.RecordCircuitBreakerMetrics(cb.name, int(state), success)
	}
}

// onSuccess is called on successful request
func (cb *CircuitBreaker) onSuccess(state CircuitBreakerState, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.OnSuccess()
	case StateHalfOpen:
		cb.counts.OnSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure is called on failed request
func (cb *CircuitBreaker) onFailure(state CircuitBreakerState, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.OnFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// currentState returns the current state of the circuit breaker
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitBreakerState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(state CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	// Update metrics
	if cb.metrics != nil {
		cb.metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(state))
	}
}

// toNewGeneration creates a new generation
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.Clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns a copy of the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	metrics  *PrometheusMetrics
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(metrics *PrometheusMetrics) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		metrics:  metrics,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(name, config, cbm.metrics)
	cbm.breakers[name] = cb
	return cb
}

// Get gets an existing circuit breaker
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	return result
}

// Remove removes a circuit breaker
func (cbm *CircuitBreakerManager) Remove(name string) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	delete(cbm.breakers, name)
}

// GetStatus returns the status of all circuit breakers
func (cbm *CircuitBreakerManager) GetStatus() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	status := make(map[string]interface{})
	for name, cb := range cbm.breakers {
		status[name] = map[string]interface{}{
			"state":  cb.State().String(),
			"counts": cb.Counts(),
		}
	}
	return status
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from CircuitBreakerState, to CircuitBreakerState) {
			fmt.Printf("Circuit breaker '%s' changed from %s to %s\n", name, from.String(), to.String())
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}
}

// DatabaseCircuitBreakerConfig returns a circuit breaker configuration for database operations
func DatabaseCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from CircuitBreakerState, to CircuitBreakerState) {
			fmt.Printf("Database circuit breaker '%s' changed from %s to %s\n", name, from.String(), to.String())
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}
}

// APICircuitBreakerConfig returns a circuit breaker configuration for API operations
func APICircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 10,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from CircuitBreakerState, to CircuitBreakerState) {
			fmt.Printf("API circuit breaker '%s' changed from %s to %s\n", name, from.String(), to.String())
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}
}

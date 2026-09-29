package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"canvaslms-gui/internal/models"
)

// Canvas bulk grading doesn't report intermediate completion reliably. Poll
// status only, independently of the GUI, with bounded network retries/backoff.
type pollPolicy struct {
	initial   time.Duration
	maximum   time.Duration
	timeout   time.Duration
	maxErrors int
}

var gradePollPolicy = pollPolicy{2 * time.Second, 30 * time.Second, 15 * time.Minute, 5}

func pollProgress(parent context.Context, fetch func(context.Context) (models.BatchProgress, error), policy pollPolicy) <-chan models.BatchProgress {
	updates := make(chan models.BatchProgress, 1)
	go func() {
		defer close(updates)
		ctx, cancel := context.WithTimeout(parent, policy.timeout)
		defer cancel()
		started := time.Now()
		lastState := "queued"
		consecutiveErrors := 0
		interval := policy.initial
		send := func(p models.BatchProgress) bool {
			p.ElapsedSeconds = int64(time.Since(started).Seconds())
			select {
			case updates <- p:
				return true
			case <-parent.Done():
				return false
			}
		}
		stop := func(message string) {
			send(models.BatchProgress{WorkflowState: lastState, Message: message, PollingError: message, PollingStopped: true})
		}
		for {
			if parent.Err() != nil {
				return
			}
			if ctx.Err() != nil {
				stop("Monitoring timed out after " + policy.timeout.String() + "; check the job in Canvas")
				return
			}
			p, err := fetch(ctx)
			if parent.Err() != nil {
				return
			}
			if ctx.Err() != nil {
				stop("Monitoring timed out after " + policy.timeout.String() + "; check the job in Canvas")
				return
			}
			if err != nil {
				consecutiveErrors++
				var auth *AuthError
				var apiErr *APIError
				permanent := errors.As(err, &auth) || (errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 429 && apiErr.StatusCode != 408)
				if permanent || consecutiveErrors >= policy.maxErrors {
					stop(fmt.Sprintf("Monitoring stopped after %d polling errors: %v. Canvas may still be processing the grades", consecutiveErrors, err))
					return
				}
				if !send(models.BatchProgress{WorkflowState: lastState, PollingError: fmt.Sprintf("Network/polling error (%d/%d), retrying: %v", consecutiveErrors, policy.maxErrors, err)}) {
					return
				}
			} else {
				consecutiveErrors = 0
				switch p.WorkflowState {
				case "queued", "running", "completed", "failed":
					lastState = p.WorkflowState
				default:
					stop("Canvas returned an unknown job state: " + p.WorkflowState)
					return
				}
				if !send(p) {
					return
				}
				if p.WorkflowState == "completed" || p.WorkflowState == "failed" {
					return
				}
			}
			timer := time.NewTimer(interval)
			select {
			case <-parent.Done():
				timer.Stop()
				return
			case <-ctx.Done():
				timer.Stop()
				stop("Monitoring timed out after " + policy.timeout.String() + "; check the job in Canvas")
				return
			case <-timer.C:
			}
			interval = min(interval+interval/2, policy.maximum)
		}
	}()
	return updates
}

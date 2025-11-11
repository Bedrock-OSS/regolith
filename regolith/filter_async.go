package regolith

import (
	"fmt"
	"sync"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

type AsyncFilter struct {
	Filter
	AsyncFilters []FilterRunner `json:"asyncFilters,omitempty"`
}

func AsyncFilterFromObject(
	obj map[string]any,
	filterDefinitions map[string]FilterInstaller,
) (*AsyncFilter, error) {
	result := &AsyncFilter{}
	// asyncFilters list
	if _, ok := obj["asyncFilters"]; !ok {
		return result, burrito.WrappedErrorf(jsonPathMissingError, "asyncFilters")
	}
	filters, ok := obj["asyncFilters"].([]any)
	if !ok {
		return result, burrito.WrappedErrorf(jsonPathTypeError, "asyncFilters", "array")
	}
	// asyncFilters list items
	for i, filter := range filters {
		filter, ok := filter.(map[string]any)
		if !ok {
			return result, burrito.WrappedErrorf(
				jsonPathTypeError, fmt.Sprintf("asyncFilters->%d", i), "object")
		}
		filterRunner, err := FilterRunnerFromObjectAndDefinitions(
			filter, filterDefinitions, true)
		if err != nil {
			return result, burrito.WrapErrorf(
				err, jsonPathParseError, fmt.Sprintf("asyncFilters->%d", i))
		}
		result.AsyncFilters = append(result.AsyncFilters, filterRunner)
	}
	return result, nil
}

// run executes all subfilters of the async filter. It returns true if the
// execution was interrupted via the RunContext.
func (f *AsyncFilter) run(context RunContext) (bool, error) {
	Logger.Debugf("RunAsyncFilter...")
	// Run the filters asynchronously
	start := time.Now()
	var wg sync.WaitGroup
	type Result struct {
		interrupted bool
		err         error
	}
	results := make(chan Result, len(f.AsyncFilters))
	for filter := range f.AsyncFilters {
		wg.Go(func() {
			filter := f.AsyncFilters[filter]
			// Disabled filters are skipped
			disabled, err := filter.IsDisabled(context)
			if err != nil {
				results <- Result{
					interrupted: false,
					err:         burrito.WrapErrorf(err, "Failed to check if filter is disabled"),
				}
				return
			}
			if disabled {
				Logger.Infof("Filter \"%s\" is disabled, skipping.", filter.GetId())
				return
			}
			// Skip printing if the filter ID is empty (most likely a nested profile)
			if filter.GetId() != "" {
				Logger.Infof("Running filter %s", filter.GetId())
			}
			// Run the filter in watch mode

			interrupted, err := filter.Run(context)

			if err != nil {
				results <- Result{
					interrupted: false,
					err:         burrito.WrapErrorf(err, filterRunnerRunError, filter.GetId()),
				}
				return
			}
			if interrupted {
				results <- Result{
					interrupted: true,
					err:         nil,
				}
				return
			}
			results <- Result{
				interrupted: false,
				err:         nil,
			}
		})
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	// Collect all results, even if we encounter an error
	// This ensures we don't leave goroutines orphaned
	var firstErr error
	var wasInterrupted bool
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		if result.interrupted {
			wasInterrupted = true
		}
	}
	Logger.Debugf("Executed in %s", time.Since(start))
	// Return the first error we encountered, if any
	if firstErr != nil {
		return false, firstErr
	}
	if wasInterrupted {
		return true, nil
	}
	return false, nil
}

func (f *AsyncFilter) Run(context RunContext) (bool, error) {
	interrupted, err := f.run(context)
	if err != nil {
		return false, burrito.PassError(err)
	}
	if interrupted {
		return true, nil
	}
	return context.IsInterrupted(), nil
}

func (f *AsyncFilter) Check(context RunContext) error {
	for _, filter := range f.AsyncFilters {
		err := filter.Check(context)
		if err != nil {
			return burrito.WrapErrorf(
				err, filterRunnerCheckError, filter.GetId())
		}
	}
	return nil
}

func (f *AsyncFilter) IsUsingDataExport(dotRegolithPath string, ctx RunContext) (bool, error) {
	for i, filter := range f.AsyncFilters {
		usingDataPath, err := filter.IsUsingDataExport(dotRegolithPath, ctx)
		if err != nil {
			return false, burrito.WrapErrorf(
				err,
				"Failed to check if subfilter is using data export.\n"+
					"Subfilter: %i", i)
		}
		if usingDataPath {
			return true, nil
		}
	}
	return false, nil
}

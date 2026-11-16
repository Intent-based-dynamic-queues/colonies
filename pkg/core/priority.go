package core

import (
	"encoding/json"
	"time"
)

// PriorityTimeUnit is the displacement of one priority unit inside the stored
// key PRIORITYTIME, in nanoseconds (negative: a higher priority subtracts more).
//
// The queue no longer orders by PRIORITYTIME alone. Under that key one unit was
// worth one day of waiting, so a job one level lower that had waited more than
// a day overtook a newer higher one. Every queue path now orders by PRIORITY
// DESC, PRIORITYTIME ASC, PROCESS_ID ASC: strict priority, and within one
// priority PRIORITYTIME differs only by submission time, so it gives exact FIFO
// (it survives ResetProcess, which rewrites SUBMISSION_TIME). The key and this
// unit are kept so stored rows, the priority channel's delta update and clients
// that read prioritytime keep working.
const PriorityTimeUnit int64 = -1000000000 * 60 * 60 * 24

// ComputePriorityTime derives the ordering key from a priority and a submission
// time. Submission and every later priority update must go through this, or
// getprocess will report a priority that disagrees with the queue order.
func ComputePriorityTime(priority int, submissionTime time.Time) int64 {
	return int64(priority)*PriorityTimeUnit + submissionTime.UnixNano()
}

// PriorityUpdate is one entry of a bulk priority-channel write.
type PriorityUpdate struct {
	ProcessID string `json:"processid"`
	Priority  int    `json:"priority"`
}

// PriorityUpdateOutcome is why a single update did or did not land.
type PriorityUpdateOutcome string

const (
	PriorityUpdated     PriorityUpdateOutcome = "updated"
	PriorityNotWaiting  PriorityUpdateOutcome = "not_waiting"
	PriorityNotFound    PriorityUpdateOutcome = "not_found"
	PriorityOutOfBounds PriorityUpdateOutcome = "rejected_out_of_bounds"
)

type PriorityUpdateResult struct {
	ProcessID string                `json:"processid"`
	Outcome   PriorityUpdateOutcome `json:"outcome"`
	Priority  int                   `json:"priority"` // effective value after the call
}

func ConvertPriorityUpdateResultsToJSON(results []PriorityUpdateResult) (string, error) {
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func ConvertJSONToPriorityUpdateResults(jsonString string) ([]PriorityUpdateResult, error) {
	var results []PriorityUpdateResult

	err := json.Unmarshal([]byte(jsonString), &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

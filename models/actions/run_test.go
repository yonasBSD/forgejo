package actions

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobStatusAggregation(t *testing.T) {
	testCases := []struct {
		desc           string
		runStatuses    []Status
		expectedStatus Status
	}{
		{
			desc:           "All jobs lower than skipped - aggregate skip",
			runStatuses:    []Status{StatusUnknown, StatusSkipped},
			expectedStatus: StatusSkipped,
		},
		{
			desc:           "Job success, no fails - aggregate success",
			runStatuses:    []Status{StatusSuccess, StatusSkipped, StatusUnknown},
			expectedStatus: StatusSuccess,
		},
		{
			desc:           "Any job failure - aggregate failure",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusSkipped, StatusUnknown},
			expectedStatus: StatusFailure,
		},
		{
			desc:           "Any job cancelled - aggregate cancelled",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusUnknown},
			expectedStatus: StatusCancelled,
		},
		{
			desc:           "Any job running - aggregate running",
			runStatuses:    []Status{StatusSuccess, StatusUnknown, StatusFailure, StatusCancelled, StatusSkipped, StatusWaiting, StatusRunning},
			expectedStatus: StatusRunning,
		},
		{
			desc:           "No jobs running, jobs waiting  - aggregate waiting",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusUnknown, StatusSkipped, StatusWaiting},
			expectedStatus: StatusWaiting,
		},
		{
			desc:           "jobs blocked, jobs waiting  - aggregate waiting",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusWaiting, StatusBlocked},
			expectedStatus: StatusWaiting,
		},
		{
			desc:           "jobs blocked, jobs running and waiting  - aggregate running",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusRunning, StatusWaiting, StatusBlocked},
			expectedStatus: StatusRunning,
		},
		{
			desc:           "jobs blocked, finished or unknown  - aggregate blocked",
			runStatuses:    []Status{StatusBlocked, StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped},
			expectedStatus: StatusBlocked,
		},
		{
			desc:           "job status unknown  - aggregate unknown",
			runStatuses:    []Status{StatusUnknown},
			expectedStatus: StatusUnknown,
		},
	}
	for _, tC := range testCases {
		mockedJobs := []*ActionRunJob{}
		for _, v := range tC.runStatuses {
			mockedJobs = append(mockedJobs, &ActionRunJob{Status: v})
		}
		// Safe guard function against order dependency
		rand.Shuffle(len(mockedJobs), func(i, j int) {
			mockedJobs[i], mockedJobs[j] = mockedJobs[j], mockedJobs[i]
		})

		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expectedStatus, aggregateJobStatus(mockedJobs))
		})
	}
}

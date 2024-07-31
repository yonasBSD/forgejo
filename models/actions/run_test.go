package actions

import (
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
			desc:           "All job skipped - aggregate skip",
			runStatuses:    []Status{StatusSkipped},
			expectedStatus: StatusSkipped,
		},
		{
			desc:           "Job success, no fails - aggregate success",
			runStatuses:    []Status{StatusSuccess, StatusSkipped},
			expectedStatus: StatusSuccess,
		},
		{
			desc:           "Any job failure - aggregate failure",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusSkipped},
			expectedStatus: StatusFailure,
		},
		{
			desc:           "Any job cancelled - aggregate cancelled",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped},
			expectedStatus: StatusCancelled,
		},
		{
			desc:           "Any job running - aggregate running",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusWaiting, StatusRunning},
			expectedStatus: StatusRunning,
		},
		{
			desc:           "No jobs running, jobs waiting  - aggregate waiting",
			runStatuses:    []Status{StatusSuccess, StatusFailure, StatusCancelled, StatusSkipped, StatusWaiting},
			expectedStatus: StatusWaiting,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			mockedJobs := []*ActionRunJob{}
			for _, v := range tC.runStatuses {
				mockedJobs = append(mockedJobs, &ActionRunJob{Status: v})
			}

			assert.Equal(t, tC.expectedStatus, aggregateJobStatus(mockedJobs))
		})
	}
}

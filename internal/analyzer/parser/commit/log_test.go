package commit

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

func Test_GivenCommitLogEntry_WhenMultipleSignersFormats_ThenUnmarshalSucceeds(t *testing.T) {
	jsonData1 := `{
		"T": "2024-09-27T05:48:39.348Z",
		"round": 1,
		"duty_id": "1234",
		"M": "Commit message",
		"commit_signers": [1, 2, 3]
	}`

	jsonData2 := `{
		"T": "2024-09-27T05:48:39.348Z",
		"round": 1,
		"duty_id": "1234",
		"M": "Commit message",
		"commit-signers": [4, 5, 6]
	}`

	timestamp := time.Date(2024, 9, 27, 5, 48, 39, 348000000, time.UTC)
	expectedTime := parser.MultiFormatTime{Time: timestamp}

	tests := []struct {
		name        string
		input       string
		expected    commitLogEntry
		expectedErr bool
	}{
		{
			name:  "Test with commit_signers field",
			input: jsonData1,
			expected: commitLogEntry{
				Timestamp:     expectedTime,
				Round:         1,
				DutyID:        "1234",
				Message:       "Commit message",
				CommitSigners: []parser.SignerID{1, 2, 3},
			},
			expectedErr: false,
		},
		{
			name:  "Test with commit-signers field",
			input: jsonData2,
			expected: commitLogEntry{
				Timestamp:     expectedTime,
				Round:         1,
				DutyID:        "1234",
				Message:       "Commit message",
				CommitSigners: []parser.SignerID{4, 5, 6},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entry commitLogEntry
			err := json.Unmarshal([]byte(tt.input), &entry)
			if (err != nil) != tt.expectedErr {
				t.Errorf("Unexpected error: got %v, wantErr %v", err, tt.expectedErr)
				return
			}

			if !reflect.DeepEqual(entry, tt.expected) {
				t.Errorf("Unmarshaled entry does not match expected.\nGot: %+v\nExpected: %+v", entry, tt.expected)
			}
		})
	}
}

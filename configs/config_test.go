package configs

import (
	"strings"
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name, input, want, errMsg string
		wantErr                   bool
	}{
		{
			name:    "Remove query parameters",
			input:   "http://example.com/path?query=123",
			want:    "http://example.com/path",
			wantErr: false,
		},
		{
			name:    "Keep URL path",
			input:   "http://example.com/path/",
			want:    "http://example.com/path",
			wantErr: false,
		},
		{
			name:    "Valid URL with https",
			input:   "https://example.com",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "URL with no scheme",
			input:   "example.com",
			want:    "",
			wantErr: true,
			errMsg:  "scheme was empty",
		},
		{
			name:    "Remove trailing slash",
			input:   "http://example.com/",
			want:    "http://example.com",
			wantErr: false,
		},
		{
			name:    "URL with no host",
			input:   "http://",
			want:    "",
			wantErr: true,
			errMsg:  "host was empty",
		},
		{
			name:    "Invalid URL format",
			input:   "://example.com",
			want:    "",
			wantErr: true,
			errMsg:  "missing protocol scheme",
		},
		{
			name:    "URL with user info",
			input:   "https://user:pass@example.com",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "Empty URL",
			input:   "",
			want:    "",
			wantErr: true,
			errMsg:  "scheme was empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("actual error = '%v', wantErrMsg: '%v'", err, tt.errMsg)
				}
			}

			if got != tt.want {
				t.Errorf("sanitizeURL() URL got = %v, want %v", got, tt.want)
			}
		})
	}
}

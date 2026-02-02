package core

import (
	"reflect"
	"testing"
)

func TestTranslatePattern(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedPattern   string
		expectedWildcards map[string]bool
	}{
		{
			name:              "simple wildcard",
			input:             "/{bucket}/{key:.*}",
			expectedPattern:   "/{bucket}/*",
			expectedWildcards: map[string]bool{"key": true},
		},
		{
			name:              "wildcard with trailing path",
			input:             "/files/{path:.*}",
			expectedPattern:   "/files/*",
			expectedWildcards: map[string]bool{"path": true},
		},
		{
			name:              "no wildcard",
			input:             "/users/{id}",
			expectedPattern:   "/users/{id}",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "multiple params with one wildcard",
			input:             "/{bucket}/{folder}/{file:.*}",
			expectedPattern:   "/{bucket}/{folder}/*",
			expectedWildcards: map[string]bool{"file": true},
		},
		{
			name:              "wildcard at root",
			input:             "/{path:.*}",
			expectedPattern:   "/*",
			expectedWildcards: map[string]bool{"path": true},
		},
		{
			name:              "no parameters",
			input:             "/static/path",
			expectedPattern:   "/static/path",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "wildcard without trailing slash",
			input:             "/prefix{key:.*}",
			expectedPattern:   "/prefix/*",
			expectedWildcards: map[string]bool{"key": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPattern, gotWildcards := TranslatePattern(tt.input)

			if gotPattern != tt.expectedPattern {
				t.Errorf("TranslatePattern() pattern = %q, want %q", gotPattern, tt.expectedPattern)
			}

			if !reflect.DeepEqual(gotWildcards, tt.expectedWildcards) {
				t.Errorf("TranslatePattern() wildcards = %v, want %v", gotWildcards, tt.expectedWildcards)
			}
		})
	}
}

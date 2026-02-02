package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTranslatePatternEdgeCases tests edge cases that gremlins identified
func TestTranslatePatternEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedPattern   string
		expectedWildcards map[string]bool
	}{
		{
			name:              "malformed - no closing brace",
			input:             "/{bucket/{key:.*}",
			expectedPattern:   "/*",
			expectedWildcards: map[string]bool{"bucket/{key": true},
		},
		{
			name:              "malformed - no opening brace for wildcard",
			input:             "/bucket/key:.*}",
			expectedPattern:   "/bucket/key:.*}",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "empty pattern",
			input:             "",
			expectedPattern:   "",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "single char pattern",
			input:             "/",
			expectedPattern:   "/",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "pattern with only opening brace",
			input:             "/{",
			expectedPattern:   "/{",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "pattern with only closing brace",
			input:             "/}",
			expectedPattern:   "/}",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "wildcard at exact end of string",
			input:             "/a{b:.*}",
			expectedPattern:   "/a/*",
			expectedWildcards: map[string]bool{"b": true},
		},
		{
			name:              "multiple regular params before wildcard",
			input:             "/{a}/{b}/{c}/{d:.*}",
			expectedPattern:   "/{a}/{b}/{c}/*",
			expectedWildcards: map[string]bool{"d": true},
		},
		{
			name:              "param without wildcard but with colon",
			input:             "/{key:pattern}",
			expectedPattern:   "/{key:pattern}",
			expectedWildcards: map[string]bool{},
		},
		{
			name:              "nested braces (malformed)",
			input:             "/{{key:.*}}",
			expectedPattern:   "/*",
			expectedWildcards: map[string]bool{"{key": true},
		},
		{
			name:              "wildcard with empty param name",
			input:             "/{:.*}",
			expectedPattern:   "/*",
			expectedWildcards: map[string]bool{"": true},
		},
		{
			name:              "param at very start",
			input:             "{key:.*}",
			expectedPattern:   "/*",
			expectedWildcards: map[string]bool{"key": true},
		},
		{
			name:              "consecutive braces",
			input:             "/{}{key:.*}",
			expectedPattern:   "/{}/*",
			expectedWildcards: map[string]bool{"key": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPattern, gotWildcards := TranslatePattern(tt.input)

			assert.Equal(t, tt.expectedPattern, gotPattern, "pattern mismatch")
			assert.Equal(t, tt.expectedWildcards, gotWildcards, "wildcards mismatch")
		})
	}
}

// TestTranslatePatternIndexBoundaries tests specific index boundary conditions
// that gremlins identified as LIVED mutations (line 37:16 and 59:18)
func TestTranslatePatternIndexBoundaries(t *testing.T) {
	t.Run("pattern ending exactly at opening brace", func(t *testing.T) {
		// This tests the condition when idx+1 would be at the end
		pattern, wildcards := TranslatePattern("/{")
		assert.Equal(t, "/{", pattern)
		assert.Empty(t, wildcards)
	})

	t.Run("pattern with param at exact end", func(t *testing.T) {
		// Tests endIdx + 1 boundary
		pattern, wildcards := TranslatePattern("/{a}")
		assert.Equal(t, "/{a}", pattern)
		assert.Empty(t, wildcards)
	})

	t.Run("wildcard immediately after param", func(t *testing.T) {
		// Tests start = endIdx + 1 boundary when endIdx is near end
		pattern, wildcards := TranslatePattern("/{a}{b:.*}")
		assert.Equal(t, "/{a}/*", pattern)
		assert.Equal(t, map[string]bool{"b": true}, wildcards)
	})

	t.Run("param at index 0", func(t *testing.T) {
		// Tests when start = 0 and idx = 0
		pattern, wildcards := TranslatePattern("{key:.*}")
		assert.Equal(t, "/*", pattern)
		assert.Equal(t, map[string]bool{"key": true}, wildcards)
	})
}

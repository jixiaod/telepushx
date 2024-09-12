package test

import (
	"telepushx/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonText(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{

		{
			name:     "Single word",
			input:    "Hello",
			expected: "Hello",
		},
		{
			name:     "Multiple words",
			input:    "Hello World",
			expected: "Hello World",
		},

		{
			name:     "Convert to html case",
			input:    "<p>🤩🤩🤩越南妹子全线升级莞式服务！！！加量不加价！！！</p>",
			expected: "🤩🤩🤩越南妹子全线升级莞式服务！！！加量不加价！！！",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := common.Text(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

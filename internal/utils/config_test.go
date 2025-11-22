package utils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	var err error
	var config ConfigOptions

	err = LoadConfig(filepath.Join("..", "..", "test", "data", "config", "config1"))
	assert.Nil(t, err)
	config = GetConfig()
	assert.Equal(t, config.Indent, 8)
	assert.Equal(t, config.NoColor, true)

	err = LoadConfig(filepath.Join("..", "..", "test", "data", "config", "config2"))
	assert.Nil(t, err)
	config = GetConfig()
	assert.Equal(t, config.Indent, 2)
}

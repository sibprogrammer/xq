package utils

import (
	"github.com/stretchr/testify/assert"
	"path"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	var err error
	var config ConfigOptions

	err = LoadConfig(path.Join("..", "..", "test", "data", "config", "config1"))
	assert.Nil(t, err)
	config = GetConfig()
	assert.Equal(t, config.Indent, 8)
	assert.Equal(t, config.NoColor, true)

	err = LoadConfig(path.Join("..", "..", "test", "data", "config", "config2"))
	assert.Nil(t, err)
	config = GetConfig()
	assert.Equal(t, config.Indent, 2)
}

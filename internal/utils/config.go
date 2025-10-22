package utils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type ConfigOptions struct {
	Indent  int
	Tab     bool
	NoColor bool
	Color   bool
	Html    bool
	Node    bool
}

var config ConfigOptions

func LoadConfig(fileName string) error {
	config.Indent = 2
	config.Tab = false
	config.NoColor = false
	config.Color = false
	config.Html = false
	config.Node = false

	file, err := os.Open(fileName)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var text = scanner.Text()
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "#") || len(text) == 0 {
			continue
		}
		var parts = strings.Split(text, "=")
		if len(parts) != 2 {
			continue
		}
		option, value := parts[0], parts[1]
		option = strings.TrimSpace(option)
		value = strings.TrimSpace(value)

		switch option {
		case "indent":
			config.Indent, _ = strconv.Atoi(value)
		case "tab":
			config.Tab, _ = strconv.ParseBool(value)
		case "no-color":
			config.NoColor, _ = strconv.ParseBool(value)
		case "color":
			config.Color, _ = strconv.ParseBool(value)
		default:
			// Ignore unknown config options for forward compatibility
		}
	}

	return nil
}

func GetConfig() ConfigOptions {
	return config
}

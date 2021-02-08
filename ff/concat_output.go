package ff

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (c *concat) computeOutput() (string, error) {
	output := c.Config.Output
	if output == "stdout" {
		output = "-"
	}
	if output != "-" && output != "" {
		abs, err := filepath.Abs(output)
		if err != nil {
			return "", fmt.Errorf("Failed to get abs path: %s", output)
		}
		output = abs
	}
	return output, nil
}

func defaultOutput(files mediaFiles, m *metadata) string {
	//default filename
	output := ""
	if m.author != "" && m.title != "" {
		output = m.author + " - " + m.title + "." + m.Config.OutputType
	} else {
		ext := filepath.Ext(files[0].Ext)
		output = strings.TrimSuffix(filepath.Base(files[0].Name), ext) + ".m4a"
	}
	output = strings.Replace(output, "/", " ", -1)
	return output
}

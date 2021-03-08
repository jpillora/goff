package ff

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (c *concat) computeOutput(files mediaFiles, m *metadata) (string, error) {
	output := c.Config.Output
	if output == "stdout" {
		output = "-"
	}
	//default filename
	if output == "" {
		if m.author != "" && m.title != "" {
			output = m.author + " - " + m.title + "." + m.Config.OutputType
		} else {
			ext := filepath.Ext(files[0].Ext)
			output = strings.TrimSuffix(filepath.Base(files[0].Name), ext) + ".m4a"
		}
	}
	//slashes not allowed
	output = strings.Replace(output, "/", " ", -1)
	//must be abs
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("Failed to get abs path: %s", output)
	}
	output = abs
	return output, nil
}

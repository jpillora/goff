package ff

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jpillora/longestcommon"
)

func (c *concat) computeInputs(files mediaFiles) error {
	//compute input paths to all audio files to concat
	inputPaths := make([]string, len(files))
	for i, f := range files {
		inputPaths[i] = f.Path
	}
	allPaths := inputPaths
	if c.Config.Output != "-" {
		allPaths = append(inputPaths, c.Config.Output)
	}
	//mount the common path across all files
	common := longestcommon.Prefix(allPaths)
	i := strings.LastIndex(common, "/")
	if i == -1 {
		log.Panicf("common: no slash in '%s'???", common)
	}
	mountDir := common[:i+1]
	if mountDir == "" {
		return fmt.Errorf("Files have no common dir")
	}
	if s, err := os.Stat(mountDir); err != nil || !s.IsDir() {
		log.Panicf("common: not a dir???")
	}
	for i := range files {
		ip := inputPaths[i]
		inputPaths[i] = strings.TrimPrefix(ip, mountDir)
		files[i].InputPath = ip
	}
	c.inputs = inputPaths
	c.logf("Input files #%d under '%s'", len(inputPaths), mountDir)
	return nil
}

func (c *concat) computeOutput(files mediaFiles, m *metadata) error {
	output := c.Config.Output
	if output == "stdout" {
		output = "-"
	}
	if output == "" {
		//default filename
		if m.author != "" && m.title != "" {
			output = m.author + " - " + m.title + "." + m.Config.OutputType
		} else {
			ext := filepath.Ext(files[0].Ext)
			output = strings.TrimSuffix(filepath.Base(files[0].Name), ext) + ".m4a"
		}
		output = strings.Replace(output, "/", " ", -1)
	}
	var err error
	if output != "-" {
		output, err = filepath.Abs(output)
		if err != nil {
			return fmt.Errorf("Failed to get abs path: %s", output)
		}
	}
	c.output = output
	c.logf("Output to '%s'", c.output)
	return nil
}

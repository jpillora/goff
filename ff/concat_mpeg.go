package ff

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/jpillora/longestcommon"
)

func (c *concat) ffmpegExec(m *metadata, files mediaFiles, output string) error {
	//input files
	inputs := []string{}
	for _, f := range files {
		inputs = append(inputs, f.Path)
	}

	//compute common dir
	mountDir, err := baseDir(append(inputs, output))
	if err != nil {
		return err
	}
	if len(mountDir) < 2 {
		return fmt.Errorf("invalid mount dir: %s", mountDir)
	}
	for i := range inputs {
		inputs[i] = strings.TrimPrefix(inputs[i], mountDir)
	}
	c.debugf("Working directory: %s", mountDir)
	//write computed metadata to disk
	metadataFile := filepath.Join(mountDir, "metadata.txt")
	if err := ioutil.WriteFile(metadataFile, m.contents.Bytes(), 0666); err != nil {
		return fmt.Errorf("Failed to write metadata file")
	}
	defer os.Remove(metadataFile)
	//compute ffmpeg args
	ff := []string{
		"-hide_banner",
		"-loglevel", "verbose",
		"-i", "concat:" + strings.Join(inputs, "|"),
		"-i", "metadata.txt", "-map_metadata", "1",
		"-vn", "-c:a", "aac", //libfdk_
		"-profile:a", "aac_he_v2",
		"-b:a", strconv.Itoa(m.bitrate) + "k",
		"-ac", "2",
		"-movflags", "+faststart",
	}
	if c.Windows {
		ff = append(ff, "-id3v2_version", "3", "-write_id3v1", "1")
	}
	if output == "-" {
		ff = append(ff, "-f", c.OutputFormat, "pipe:1")
	} else {
		relaOut := strings.TrimPrefix(output, mountDir)
		ff = append(ff, relaOut, "-y" /*yes, overwrite*/)
	}
	//compute docker args
	docker := []string{}
	if c.docker("ffmpeg") {
		docker = []string{
			"-v", fmt.Sprintf("%s:%s", mountDir, mountDir),
			"-w", mountDir,
		}
	}
	t0 := time.Now()
	cmd := c.cmd("ffmpeg", docker, ff)
	cmd.Dir = mountDir
	// cmd.Stdin = strings.NewReader(metadata.String())
	if output == "-" {
		cmd.Stdout = os.Stdout //attach stdout
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to get ffmpeg stderr: %s", err)
	}
	//progress bar
	bar := pb.StartNew(int(m.duration.Milliseconds() / 10))
	//monitor err pipe for current progress
	errbytes := bytes.Buffer{}
	go func() {
		buff := make([]byte, 4096)
		for {
			n, err := stderr.Read(buff)
			if err != nil {
				break
			}
			b := buff[:n]
			errbytes.Write(b)
			// time=00:01:39.28
			m := timeRe.FindStringSubmatch(string(b))
			if len(m) == 0 {
				continue
			}
			d := time.Duration(mustInt(m[1]))*time.Hour +
				time.Duration(mustInt(m[2]))*time.Minute +
				time.Duration(mustInt(m[3]))*time.Second +
				time.Duration(mustInt(m[4])*10)*time.Millisecond
			currMilli := d.Nanoseconds() / 1e6
			bar.Set64(currMilli / 10)
		}
	}()
	c.debugf("Metadata:\n\n%s\n\n", m.contents.String())
	//start
	bar.ShowFinalTime = true
	bar.ShowCounters = false
	bar.Update()
	if err := cmd.Run(); err != nil {
		if errbytes.Len() > 0 {
			err = errors.New(errbytes.String())
		}
		return fmt.Errorf("Failed to run ffmpeg: %s", err)
	}
	bar.FinishPrint("Done in " + time.Now().Sub(t0).String())
	c.debugf("Error out: %s", errbytes.String())
	return nil
}

func baseDir(paths []string) (dir string, err error) {
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			panic("all paths must be abs")
		}
	}
	common := longestcommon.Prefix(paths)
	if len(common) < 2 {
		return "", errors.New("no common dir")
	}
	i := strings.LastIndex(common, "/")
	if i == -1 {
		return "", fmt.Errorf("common: no slash in '%s'???", common)
	}
	mountDir := common[:i+1]
	if mountDir == "" {
		return "", fmt.Errorf("Files have no common dir")
	}
	if s, err := os.Stat(mountDir); err != nil || !s.IsDir() {
		return "", fmt.Errorf("common dir not a dir")
	}
	return mountDir, nil
}

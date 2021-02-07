package ff

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
)

func (c *concat) ffmpegArgs(m *metadata) []string {

	args := []string{}

	if c.docker("ffmpeg") {
		args = append(args,
			"run", "--rm",
			"-v", fmt.Sprintf("%s:%s", c.mountDir, c.mountDir),
			"-w", c.mountDir,
			"--entrypoint", "ffmpeg",
			c.image(),
		)
	}

	args = append(args,
		"-hide_banner",
		"-loglevel", "verbose",
		"-i", "concat:"+strings.Join(c.inputs, "|"),
		"-i", c.metadataFile(), "-map_metadata", "1",
		"-vn", "-c:a", "libfdk_aac",
		"-profile:a", "aac_he_v2",
		"-b:a", strconv.Itoa(m.bitrate)+"k",
		"-ac", "2",
	)
	if c.Windows {
		args = append(args, "-id3v2_version", "3", "-write_id3v1", "1")
	}
	if c.output == "-" {
		args = append(args, "-f", c.OutputFormat, "pipe:1")
	} else {
		relaOut := strings.TrimPrefix(c.output, c.mountDir)
		args = append(args, relaOut, "-y" /*yes, overwrite*/)
	}
	return args
}

func (c *concat) ffmpegExec(m *metadata) error {
	if err := ioutil.WriteFile(c.metadataFile(), m.contents.Bytes(), 0666); err != nil {
		return fmt.Errorf("Failed to write metadata file")
	}
	t0 := time.Now()
	//progress bar
	bar := pb.StartNew(int(m.duration.Milliseconds() / 10))
	bar.ShowFinalTime = true
	bar.ShowCounters = false
	bar.Update()

	cmd := c.cmd("ffmpeg", c.ffmpegArgs(m)...)
	cmd.Dir = os.TempDir()
	// cmd.Stdin = strings.NewReader(metadata.String())
	if c.output == "-" {
		cmd.Stdout = os.Stdout //attach stdout
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to get ffmpeg stderr: %s", err)
	}
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

	if c.Debug {
		log.Printf("Metadata:\n\n%s\n\n", m.contents.String())
		log.Printf("Execute: %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	}

	if err := cmd.Run(); err != nil {
		if errbytes.Len() > 0 {
			err = errors.New(errbytes.String())
		}
		return fmt.Errorf("Failed to run ffmpeg: %s", err)
	}
	bar.FinishPrint("Done in " + time.Now().Sub(t0).String())
	if c.Debug {
		log.Printf("Error out: %s", errbytes.String())
	}

	return nil

}

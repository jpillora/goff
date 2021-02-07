package ff

import (
	"errors"
	"fmt"
	"log"
)

type Config struct {
	Inputs       []string `opts:"mode=arg, min=1, help=inputs are audio files and directories of audio files"`
	Output       string   `help:"Output file (defaults to <input>.m4a)"`
	OutputFormat string   `help:"When output is 'stdout', output file format determines encoder"`
	OutputType   string   `help:"When output is empty, output file is '<author> - <title>.<output type>'"`
	MaxBitrate   int      `help:"Bitrate in KB/s (when source bitrate is higher)"`
	NoStderr     bool     `help:"Detach stderr"`
	Windows      bool     `help:"ID3 Windows support"`
	Debug        bool     `help:"Show debug output"`
	Docker       bool     `help:"Use docker even if ffmpeg installed locally"`
}

func Concat(c Config) error {
	m := concat{Config: c}
	return m.Concat()
}

type concat struct {
	Config
	//
	output   string
	mountDir string
	inputs   []string
}

func (c *concat) Concat() error {
	if len(c.Inputs) == 0 {
		return errors.New("No input files provided")
	}
	count := new(int)
	files := mediaFiles{}
	for _, path := range c.Inputs {
		if err := c.addPath(&files, path, count); err != nil {
			return fmt.Errorf("Failed to add path: %s: %s", path, err)
		}
	}
	if len(files) == 0 {
		return errors.New("No audio files provided")
	}
	if err := c.computeInputs(files); err != nil {
		return err
	}
	if err := c.probeMediaFiles(files); err != nil {
		return err
	}

	m := &metadata{Config: c.Config}
	if err := m.computeHeader(files); err != nil {
		return err
	}
	if err := m.computeBitrate(files); err != nil {
		return err
	}
	if err := c.computeOutput(files, m); err != nil {
		return err
	}

	c.logf("Input '%s' by '%s' (#%d tracks, %s total, bitrate %dk -> %dk)",
		m.title, m.author, len(files), m.duration, m.fileBitrate, m.bitrate)
	return nil

}

func (c *concat) logf(format string, args ...interface{}) {
	log.Printf("[goff] "+format, args...)
}

func (c *concat) debugf(format string, args ...interface{}) {
	if c.Config.Debug {
		c.logf("[DEBUG] "+format, args...)
	}
}

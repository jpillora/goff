package ff

import (
	"os/exec"
	"path/filepath"
)

func (m concat) image() string {
	return "linuxserver/ffmpeg"
}

//use docker to run prog?
func (c *concat) docker(prog string) bool {
	if c.Config.Docker {
		return true
	}
	_, err := exec.LookPath(prog)
	return err == nil
}

func (c *concat) metadataFile() string {
	return filepath.Join(c.mountDir, "metadata.txt")
}

func (c *concat) cmd(prog string, args ...string) *exec.Cmd {
	if prog != "ffmpeg" && prog != "ffprobe" {
		panic("unknown prog")
	}
	if c.docker(prog) {
		prog = "docker"
	}
	cmd := exec.Command(prog, args...)
	return cmd, nil
}

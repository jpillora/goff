package ff

import (
	"os/exec"
	"strings"
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
	installed := err == nil
	return !installed
}

func (c *concat) cmd(prog string, dockerArgs, ffArgs []string) *exec.Cmd {
	if prog != "ffmpeg" && prog != "ffprobe" {
		panic("unknown prog")
	}
	args := ffArgs
	if c.docker(prog) {
		d := []string{
			"run", "--rm",
			"--entrypoint", prog,
		}
		d = append(d, dockerArgs...)
		d = append(d, c.image())
		//convert to docker args
		args = append(d, args...)
		prog = "docker"
	}
	c.debugf("Command: %s %s", prog, strings.Join(args, " "))
	cmd := exec.Command(prog, args...)
	return cmd
}

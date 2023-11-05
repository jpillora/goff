package ff

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type mediaFiles []*mediaFile

func (c concat) addPath(files *mediaFiles, path string, n *int) error {
	(*n)++
	if *n > 1000 {
		return errors.New("Too many files")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	//recurse into dirs
	if info.IsDir() {
		c.debugf("add directory: %s", path)
		infos, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}
		for _, info := range infos {
			p := filepath.Join(path, info.Name())
			if err := c.addPath(files, p, n); err != nil {
				return err
			}
		}
		return nil
	}
	ext := getAudioExt(path)
	if ext == "" {
		return nil
	}
	c.debugf("add audio file: %s", path)
	//parse info
	f := &mediaFile{}
	f.Path = path
	f.Name = strings.TrimSuffix(info.Name(), ext)
	f.Ext = ext
	//add file!
	*files = append(*files, f)
	return nil
}

// sort interface for mediaFiles
func (files mediaFiles) Len() int {
	return len(files)
}

func (files mediaFiles) Less(i, j int) bool {
	return padNums(files[i].Name) < padNums(files[j].Name)
}
func (files mediaFiles) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

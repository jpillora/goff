package ff

import (
	"bytes"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jpillora/longestcommon"
)

const timebase = "1/1000" //milliseconds

type metadata struct {
	Config
	//computed from files:
	contents    bytes.Buffer
	title       string
	author      string
	fileBitrate int
	bitrate     int
	duration    time.Duration
}

func newMetadata(c Config, files mediaFiles) (*metadata, error) {
	m := &metadata{Config: c}
	if err := m.computeHeader(files); err != nil {
		return nil, err
	}
	if err := m.computeBitrate(files); err != nil {
		return nil, err
	}
	if err := m.computeChapters(files); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *metadata) computeHeader(files mediaFiles) error {
	title := strSet{}
	album := strSet{}
	artist := strSet{}
	genre := strSet{}
	date := strSet{}
	totalMilli := 0
	for _, f := range files {
		if !f.probed {
			return errors.New("file must be probed")
		}
		title.Add(f.Title)
		//sum duration
		totalMilli += f.probe.Format.Duration
		//get tags
		tags := f.probe.Format.Tags
		//find shared artist/author
		album.Add(tags.Album)
		artist.Add(tags.Artist)
		genre.Add(tags.Genre)
		date.Add(tags.Date)
	}
	//
	m.title = title.Get()
	if m.title == "" {
		m.title = album.Get()
	}
	m.author = artist.Get()
	m.duration = time.Duration(totalMilli) * time.Millisecond //timebase

	m.contents.WriteString(";FFMETADATA1\n")
	m.contents.WriteString("album=" + m.title + "\n")
	m.contents.WriteString("artist=" + m.author + "\n")
	m.contents.WriteString("title=" + m.title + "\n")
	m.contents.WriteString("genre=" + genre.Get() + "\n")
	m.contents.WriteString("TLEN=" + strconv.Itoa(totalMilli) + "\n")
	m.contents.WriteString("encoded_by=goff\n")
	m.contents.WriteString("date=" + date.Get() + "\n")
	return nil
}

func (m *metadata) computeBitrate(files mediaFiles) error {
	inputBitrate := 0x1fffffff //Infinity
	for _, f := range files {
		//get min bitrate
		if br, err := strconv.Atoi(f.probe.Format.BitRate); err == nil {
			if br < inputBitrate {
				inputBitrate = br
			}
		}
	}
	bitrate := inputBitrate
	if bitrate == 0 {
		bitrate = m.Config.MaxBitrate
	} else {
		//covert from bytes/s to kilobytes/s
		bitrate /= 1000
		m.fileBitrate = bitrate
	}
	if m.MaxBitrate < bitrate {
		bitrate = m.Config.MaxBitrate
	}
	m.bitrate = bitrate
	return nil
}

func (m *metadata) computeChapters(files mediaFiles) error {
	//calculate chapter names without prefix/suffixes
	chapterNames := make([]string, len(files))
	for i, f := range files {
		chapterNames[i] = f.Name
	}
	longestcommon.TrimPrefix(chapterNames)
	longestcommon.TrimSuffix(chapterNames)
	for i, f := range files {
		chapterNames[i] = strings.TrimSpace(nonAlpha.ReplaceAllString(f.Name, ""))
	}
	offset := 0
	for i, f := range files {
		log.Printf("[#%3d] %s (%s)", i+1, f.Title, time.Duration(f.probe.Format.Duration)*time.Millisecond)
		//write chapter
		m.contents.WriteString("[CHAPTER]\n")
		m.contents.WriteString("TIMEBASE=" + timebase + "\n")
		m.contents.WriteString("START=" + strconv.Itoa(offset) + "\n")
		offset += f.probe.Format.Duration
		m.contents.WriteString("END=" + strconv.Itoa(offset) + "\n")
		m.contents.WriteString("title=" + chapterNames[i] + "\n")
	}
	return nil
}

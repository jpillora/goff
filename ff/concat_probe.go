package ff

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

func (m concat) probeMediaFiles(files mediaFiles) error {
	for _, f := range files {
		if err := m.probeMediaFile(f); err != nil {
			return err
		}
	}
	return nil
}

func (c concat) probeMediaFile(f *mediaFile) error {
	c.debugf("probe audio file: %s", f.Path)
	//extract media info
	docker := []string{"-v", f.Path + ":" + f.Path}
	ff := []string{"-v", "quiet", "-show_format", "-show_streams", "-of", "json", f.Path}
	cmd := c.cmd("ffprobe", docker, ff)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			err = errors.New(string(out))
		}
		return fmt.Errorf("get file: %s: ffprobe: %s", f.Path, err)
	}
	if err := json.Unmarshal(out, &f.probe); err != nil {
		return fmt.Errorf("get file: %s: ffprobe: json: %s\n%s", f.Path, err, out)
	}
	f.probed = true
	//find title
	if f.probe.Format.Tags.Title != "" {
		f.Title = f.probe.Format.Tags.Title
	}
	//parse duration
	if f.probe.Format.DurationStr == "" {
		// return nil, fmt.Errorf("get file: %s: missing duration: %s", path, string(out))
		c.logf("[WARNING] cannot find duration for: %s", f.Path)
		f.probe.Format.Duration = 0
	} else {
		duration, err := strconv.ParseFloat(f.probe.Format.DurationStr, 64)
		if err != nil {
			return fmt.Errorf("get file: %s: parse-float: %s", f.Path, err)
		}
		f.probe.Format.Duration = int(duration * 1000)
	}
	return nil
}

//ffmpeg json outputs
type mediaFile struct {
	Path string
	Name string
	Ext  string
	//
	Title  string
	Artist string
	Album  string
	Genre  string
	//
	probe  probe
	probed bool
}

type probe struct {
	Streams []stream `json:"streams"`
	Format  struct {
		Filename       string `json:"filename"`
		NbStreams      int    `json:"nb_streams"`
		NbPrograms     int    `json:"nb_programs"`
		FormatName     string `json:"format_name"`
		FormatLongName string `json:"format_long_name"`
		StartTime      string `json:"start_time"`
		DurationStr    string `json:"duration"`
		Duration       int    `json:"-"`
		Size           string `json:"size"`
		BitRate        string `json:"bit_rate"`
		ProbeScore     int    `json:"probe_score"`
		Tags           struct {
			Artist string `json:"artist"`
			Album  string `json:"album"`
			Genre  string `json:"genre"`
			Title  string `json:"title"`
			Track  string `json:"track"`
			Date   string `json:"date"`
		} `json:"tags"`
	} `json:"format"`
}

type stream struct {
	Index          int    `json:"index"`
	CodecName      string `json:"codec_name"`
	CodecLongName  string `json:"codec_long_name"`
	CodecType      string `json:"codec_type"`
	CodecTimeBase  string `json:"codec_time_base"`
	CodecTagString string `json:"codec_tag_string"`
	CodecTag       string `json:"codec_tag"`
	SampleFmt      string `json:"sample_fmt,omitempty"`
	SampleRate     string `json:"sample_rate,omitempty"`
	Channels       int    `json:"channels,omitempty"`
	ChannelLayout  string `json:"channel_layout,omitempty"`
	BitsPerSample  int    `json:"bits_per_sample,omitempty"`
	RFrameRate     string `json:"r_frame_rate"`
	AvgFrameRate   string `json:"avg_frame_rate"`
	TimeBase       string `json:"time_base"`
	StartPts       int    `json:"start_pts"`
	StartTime      string `json:"start_time"`
	DurationTs     int64  `json:"duration_ts"`
	Duration       string `json:"duration"`
	BitRate        string `json:"bit_rate,omitempty"`
	Disposition    struct {
		Default         int `json:"default"`
		Dub             int `json:"dub"`
		Original        int `json:"original"`
		Comment         int `json:"comment"`
		Lyrics          int `json:"lyrics"`
		Karaoke         int `json:"karaoke"`
		Forced          int `json:"forced"`
		HearingImpaired int `json:"hearing_impaired"`
		VisualImpaired  int `json:"visual_impaired"`
		CleanEffects    int `json:"clean_effects"`
		AttachedPic     int `json:"attached_pic"`
		TimedThumbnails int `json:"timed_thumbnails"`
	} `json:"disposition"`
	Tags struct {
		Encoder string `json:"encoder"`
	} `json:"tags"`
	SideDataList []struct {
		SideDataType string `json:"side_data_type"`
		SideDataSize int    `json:"side_data_size"`
	} `json:"side_data_list,omitempty"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	CodedWidth         int    `json:"coded_width,omitempty"`
	CodedHeight        int    `json:"coded_height,omitempty"`
	HasBFrames         int    `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	PixFmt             string `json:"pix_fmt,omitempty"`
	Level              int    `json:"level,omitempty"`
	ColorRange         string `json:"color_range,omitempty"`
	Refs               int    `json:"refs,omitempty"`
}

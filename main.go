package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/jpillora/longestcommon"
	"github.com/jpillora/opts"
)

const (
	ffmpegImage = "jrottenberg/ffmpeg"
)

//BuiltTime is the build time in unix-epoch seconds
var BuiltTime = "-"

var c = struct {
	Inputs       []string `opts:"mode=arg, min=1, help=inputs are audio files and directories of audio files"`
	Output       string   `help:"Output file (defaults to <input>.m4a)"`
	OutputFormat string   `help:"When output is 'stdout', output file format determines encoder"`
	OutputType   string   `help:"When output is empty, output file is '<author> - <title>.<output type>'"`
	MaxBitrate   int      `help:"Bitrate in KB/s (when source bitrate is higher)"`
	NoStderr     bool     `help:"Detach stderr"`
	Windows      bool     `help:"ID3 Windows support"`
	Debug        bool     `help:"Show debug output"`
}{
	OutputFormat: "adts",
	OutputType:   "m4a",
	MaxBitrate:   48,
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

var nonAlpha = regexp.MustCompile(`[^\s\w\-]`)

func run() error {

	opts.New(&c).Version(BuiltTime).Parse()

	if len(c.Inputs) == 0 {
		return errors.New("No input files provided")
	}

	output := c.Output
	if output == "stdout" {
		output = "-"
	}

	count := new(int)
	files := []*mediaFile{}
	for _, path := range c.Inputs {
		if err := addPath(&files, path, count); err != nil {
			return fmt.Errorf("Failed to add path: %s: %s", path, err)
		}
	}
	if len(files) == 0 {
		return errors.New("No audio files provided")
	}

	inputBitrate := 0x1fffffff //Infinity
	album := strSet{}
	artist := strSet{}
	genre := strSet{}
	date := strSet{}
	totalMilli := 0
	for _, f := range files {
		//get min bitrate
		if br, err := strconv.Atoi(f.Format.BitRate); err == nil {
			if br < inputBitrate {
				inputBitrate = br
			}
		}
		//sum duration
		totalMilli += f.Format.Duration
		//get tags
		tags := f.Format.Tags
		//find title
		if tags.Title != "" {
			f.Title = tags.Title
		} else {
			f.Title = f.Name
		}
		//find shared artist/author
		album.Add(tags.Album)
		artist.Add(tags.Artist)
		genre.Add(tags.Genre)
		date.Add(tags.Date)
	}

	bitrate := inputBitrate
	if bitrate == 0 {
		bitrate = c.MaxBitrate
	} else {
		//covert from bytes/s to kilobytes/s
		bitrate /= 1000
	}
	if c.MaxBitrate < bitrate {
		bitrate = c.MaxBitrate
	}

	if output == "" {
		author := artist.Get()
		title := album.Get()
		if author != "" && title != "" {
			output = author + " - " + title + "." + c.OutputType
		} else {
			ext := filepath.Ext(c.Inputs[0])
			output = strings.TrimSuffix(filepath.Base(c.Inputs[0]), ext) + ".m4a"
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

	metadata := bytes.Buffer{}
	metadata.WriteString(";FFMETADATA1\n")
	metadata.WriteString("album=" + album.Get() + "\n")
	metadata.WriteString("artist=" + artist.Get() + "\n")
	metadata.WriteString("title=" + album.Get() + "\n")
	metadata.WriteString("genre=" + genre.Get() + "\n")
	metadata.WriteString("TLEN=" + strconv.Itoa(totalMilli) + "\n")
	metadata.WriteString("encoded_by=goff\n")
	metadata.WriteString("date=" + date.Get() + "\n")

	totalDuration := time.Duration(totalMilli) * time.Millisecond
	log.Printf("Input '%s' by '%s' (#%d tracks, %s total, bitrate %dk -> %dk)", album.Get(), artist.Get(), len(files), totalDuration, inputBitrate/1000, bitrate)
	log.Printf("Output to '%s'", output)

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

	inputPaths := make([]string, len(files))
	offset := 0
	for i, f := range files {
		log.Printf("[#%3d] %s (%s)", i+1, f.Title, time.Duration(f.Format.Duration)*time.Millisecond)
		//add path
		inputPaths[i] = f.Path
		//write chapter
		metadata.WriteString("[CHAPTER]\n")
		metadata.WriteString("TIMEBASE=1/1000\n")
		metadata.WriteString("START=" + strconv.Itoa(offset) + "\n")
		offset += f.Format.Duration
		metadata.WriteString("END=" + strconv.Itoa(offset) + "\n")
		metadata.WriteString("title=" + chapterNames[i] + "\n")
	}

	allPaths := inputPaths
	if output != "-" {
		allPaths = append(inputPaths, output)
	}

	common := longestcommon.Prefix(allPaths)
	i := strings.LastIndex(common, "/")
	if i == -1 {
		log.Panicf("common: no slash at all???")
	}
	mountDir := common[:i+1]
	if mountDir == "" {
		return fmt.Errorf("Files have no common dir")
	}
	if s, err := os.Stat(mountDir); err != nil || !s.IsDir() {
		log.Panicf("common: not a dir???")
	}
	//
	dockerArgs := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:%s", mountDir, mountDir),
		"-w", mountDir,
		ffmpegImage,
	}

	for i, f := range inputPaths {
		inputPaths[i] = strings.TrimPrefix(f, mountDir)
	}

	metadataFile := filepath.Join(mountDir, "metadata.txt")
	if err := ioutil.WriteFile(metadataFile, metadata.Bytes(), 0666); err != nil {
		return fmt.Errorf("Failed to write metadata file")
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "verbose",
		"-i", "concat:" + strings.Join(inputPaths, "|"),
		"-i", metadataFile, "-map_metadata", "1",
		"-vn", "-c:a", "libfdk_aac",
		"-profile:a", "aac_he_v2",
		"-b:a", strconv.Itoa(bitrate) + "k",
		"-ac", "2",
	}

	if c.Windows {
		args = append(args, "-id3v2_version", "3", "-write_id3v1", "1")
	}
	if output == "-" {
		args = append(args, "-f", c.OutputFormat, "pipe:1")
	} else {
		args = append(args, strings.TrimPrefix(output, mountDir), "-y")
	}

	if c.Debug {
		log.Printf("Metadata:\n\n%s\n\n", metadata.String())
		log.Printf("Docker: %s", strings.Join(dockerArgs, " "))
		log.Printf("Executing: ffmpeg %s", strings.Join(args, " "))
	}

	t0 := time.Now()
	//progress bar
	bar := pb.StartNew(totalMilli / 10)
	bar.ShowFinalTime = true
	bar.ShowCounters = false
	bar.Update()
	cmd := exec.Command("docker", append(dockerArgs, args...)...)
	cmd.Dir = os.TempDir()
	// cmd.Stdin = strings.NewReader(metadata.String())
	if output == "-" {
		cmd.Stdout = os.Stdout //attach stdout
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Failed to get ffmpeg stderr: %s", err)
	}

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

func addPath(files *[]*mediaFile, path string, n *int) error {
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
		if c.Debug {
			log.Printf("[debug] add directory: %s", path)
		}
		infos, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}
		for _, info := range infos {
			p := filepath.Join(path, info.Name())
			if err := addPath(files, p, n); err != nil {
				return err
			}
		}
		return nil
	}
	ext := getAudioExt(path)
	if ext == "" {
		return nil
	}
	if c.Debug {
		log.Printf("[debug] add audio file: %s", path)
	}
	//parse info
	f, err := getMediaFile(path)
	if err != nil {
		return err
	}
	f.Name = strings.TrimSuffix(info.Name(), ext)
	//add file!
	*files = append(*files, f)
	return nil
}

func getAudioExt(path string) string {
	for _, ext := range []string{".mp3", ".m4a", ".m4b"} {
		if strings.HasSuffix(path, ext) {
			return ext
		}
	}
	return ""
}

var timeRe = regexp.MustCompile(`time=(\d\d):(\d\d):(\d\d)\.(\d\d)`)

func getMediaFile(path string) (*mediaFile, error) {
	if c.Debug {
		log.Printf("[debug] parse audio file: %s", path)
	}
	//extract media info
	cmd := exec.Command("ffprobe", "-v", "error", "-show_format", "-show_streams", "-of", "json", path)
	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			err = errors.New(string(out))
		}
		return nil, fmt.Errorf("get file: %s: ffprobe: %s", path, err)
	}
	f := &mediaFile{}
	if err := json.Unmarshal(out, f); err != nil {
		return nil, fmt.Errorf("get file: %s: ffprobe: json: %s\n%s", path, err, out)
	}
	//add more
	f.Path = path
	//
	if f.Format.DurationStr == "" {
		// return nil, fmt.Errorf("get file: %s: missing duration: %s", path, string(out))
		log.Printf("[WARNING] cannot find duration for: %s", path)
		f.Format.Duration = 0
	} else {
		duration, err := strconv.ParseFloat(f.Format.DurationStr, 64)
		if err != nil {
			return nil, fmt.Errorf("get file: %s: parse-float: %s", path, err)
		}
		f.Format.Duration = int(duration * 1000)
	}
	return f, nil
}

func mustInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

//string set
type strSet struct {
	strings map[string]int
}

func (set *strSet) Add(s string) {
	if set.strings == nil {
		set.strings = map[string]int{}
	}
	set.strings[s] = set.strings[s] + 1
}

func (set *strSet) Get() string {
	n := -1
	s := ""
	for k, v := range set.strings {
		if v > n {
			n = v
			s = k
		}
	}
	return s
}

//ffmpeg json outputs
type mediaFile struct {
	Path string `json:"-"`
	Name string `json:"-"`
	//
	Title  string `json:"-"`
	Artist string `json:"-"`
	Album  string `json:"-"`
	Genre  string `json:"-"`
	//
	Streams []mediaStream `json:"streams"`
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

type mediaStream struct {
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

# goff

Concatenates a provided set of audio files into a single file. Uses local `ffmpeg`, otherwise uses containerised ffmpeg using `docker`.

```
$ goff --help

  Usage: goff [options] <input> [input] ...

  inputs are audio files and directories of audio files

  Options:
  --output, -o       Output file (defaults to <input>.m4a)
  --output-format    When output is 'stdout', output file format determines encoder (default adts)
  --output-type      When output is empty, output file is '<author> - <title>.<output type>' (default
                     m4a)
  --max-bitrate, -m  Bitrate in KB/s (when source bitrate is higher, default 64)
  --no-stderr, -n    Detach stderr
  --windows, -w      ID3 Windows support
  --debug, -d        Show debug output
  --docker           Use docker even if ffmpeg installed locally
  --version, -v      display version
  --help, -h         display help

  Version:
    0.0.0

```
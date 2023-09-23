package zlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
}
func ConsoleLogger() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("|%-5s|", i))
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
	}
	return output
}
func Default() {
	Init()
	log.Logger = log.Output(ConsoleLogger())
}

func SetOutput(output io.Writer) {
	multi := zerolog.MultiLevelWriter(ConsoleLogger(), output)
	log.Logger = log.Output(multi)
}

type RotatingFile struct {
	file        *os.File
	fileName    string
	maxBytes    int64
	backupCount int
}

func NewRotatingFile(fileName string, bytes int64, backupCount int) (*RotatingFile, error) {
	if bytes <= 0 {
		return nil, errors.New("bytes must be greater than 0")
	}
	filePath := path.Dir(fileName)
	if err := os.MkdirAll(filePath, 0666); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	rf := &RotatingFile{
		file:        file,
		fileName:    fileName,
		maxBytes:    bytes,
		backupCount: backupCount,
	}
	return rf, nil
}

func (rf *RotatingFile) Write(p []byte) (n int, err error) {
	rf.rotating(int64(len(p)))
	return rf.file.Write(p)
}

func (rf *RotatingFile) Close() error {
	return rf.file.Close()
}

func (rf *RotatingFile) rotating(wn int64) {
	if rf.backupCount < 1 {
		return
	}
	if fileInfo, err := rf.file.Stat(); err != nil {
		return
	} else if fileInfo.Size()+wn < rf.maxBytes {
		return
	}

	var oldPath, newPath string
	for i := rf.backupCount - 1; i > 0; i-- {
		oldPath = fmt.Sprintf("%s-%d", rf.fileName, i)
		newPath = fmt.Sprintf("%s-%d", rf.fileName, i+1)
		os.Rename(oldPath, newPath)
	}
	rf.file.Sync()
	rf.file.Close()
	newPath = rf.fileName + "-1"
	os.Rename(rf.fileName, newPath)
	rf.file, _ = os.OpenFile(rf.fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

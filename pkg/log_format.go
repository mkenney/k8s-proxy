/*
This file defines a custom log format in the form of:
[ISO-8601-date] [level] [hostname] [caller] [message]

	log.SetFormatter(&textFormat{})
	log.SetFormatter(&jsonFormat{})
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

/*
RFC3339Milli defines an RFC3339 date format with miliseconds
*/
const RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

/*
Set the formatter and the default level
The level is defined by the LOG_LEVEL environment variable. Default is 'info'
*/
func init() {
	levelFlag := os.Getenv("LOG_LEVEL")
	if "" == levelFlag {
		levelFlag = "info"
	}

	level, err := log.ParseLevel(levelFlag)
	if nil != err {
		log.Fatalf("Could not parse log level flag: %s", err.Error())
	}

	log.SetFormatter(&textFormat{})
	log.SetLevel(level)
}

func getCaller() string {
	caller := ""
	a := 0
	for {
		if pc, file, line, ok := runtime.Caller(a + 2); ok {
			if !strings.Contains(strings.ToLower(file), "github.com/sirupsen/logrus") {
				caller = strings.Replace(fmt.Sprintf("%s:%d %s", path.Base(file), line, runtime.FuncForPC(pc).Name()), "github.com/mkenney/k8s-proxy", "", -1)
				break
			}
		} else {
			break
		}
		a++
	}
	return caller
}

type logData struct {
	Timestamp string      `json:"time"`
	Level     string      `json:"level"`
	Hostname  string      `json:"host"`
	Caller    string      `json:"caller"`
	Message   string      `json:"msg"`
	Data      []dataField `json:"data"`
}
type dataField struct {
	Key string
	Msg string
}

type jsonFormat struct {
	*log.JSONFormatter
}

/*
Format is a custom log format method
*/
func (l *jsonFormat) Format(entry *log.Entry) ([]byte, error) {
	data := getData(entry)
	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal log data as JSON: %s", err.Error())
	}
	return append(serialized, '\n'), nil
}

type textFormat struct {
	*log.TextFormatter
}

/*
Format is a custom log format method
*/
func (l *textFormat) Format(entry *log.Entry) ([]byte, error) {
	var logLine *bytes.Buffer

	if entry.Buffer != nil {
		logLine = entry.Buffer
	} else {
		logLine = &bytes.Buffer{}
	}

	data := getData(entry)
	textTemplate.Execute(logLine, data)
	logLine.WriteByte('\n')
	return logLine.Bytes(), nil
}

var textTemplate = template.Must(
	template.New("log").Parse(`time="{{.Timestamp}}" host="{{.Hostname}}" level="{{.Level}}" caller="{{.Caller}}" msg="{{.Message}}" {{range $k, $v := .Data}}{{$v.Key}}="{{$v.Msg}}" {{end}}`),
)

/*
getData is a helper function that extracts log data from the logrus
entry.
*/
func getData(entry *log.Entry) *logData {
	data := &logData{
		Timestamp: entry.Time.Format(RFC3339Milli),
		Level:     entry.Level.String(),
		Hostname:  os.Getenv("HOSTNAME"),
		Caller:    getCaller(),
		Message:   entry.Message,
		Data:      make([]dataField, 0),
	}

	keys := make([]string, 0)
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		data.Data = append(data.Data, dataField{
			Key: v,
			Msg: fmt.Sprintf("%v", entry.Data[v]),
		})
	}

	return data
}

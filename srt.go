package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
The timecode format used is hours:minutes:seconds,milliseconds with time units
fixed to two zero-padded digits and fractions fixed to three zero-padded digits
(00:00:00,000). The fractional separator used is the comma.
*/
type TimeCode struct {
	time time.Time
}

func parseTimeCodeStr(str string) (hour, min, sec, msec int) {
	hour, _ = strconv.Atoi(str[0:2])
	min, _ = strconv.Atoi(str[3:5])
	sec, _ = strconv.Atoi(str[6:8])
	msec, _ = strconv.Atoi(str[9:12])
	return
}

func Str2TimeCode(str string, t *TimeCode) {
	hour, min, sec, msec := parseTimeCodeStr(str)
	nanosec := msec * 1000 * 1000
	t.time = time.Date(2000, time.April, 1, hour, min, sec, nanosec, time.UTC)
}

func (t TimeCode) String() string {
	str := t.time.Format("15:04:05.000")
	return str[0:8] + "," + str[9:]
}

func (t *TimeCode) Add(str string) error {
	hour, min, sec, msec := parseTimeCodeStr(str)
	dstr := fmt.Sprintf("%dh%dm%ds%dms", hour, min, sec, msec)
	d, err := time.ParseDuration(dstr)
	if err != nil {
		fmt.Println("parse duration:", err)
		return err
	}
	t.time = t.time.Add(d)
	return nil
}

type SrtItem struct {
	Counter   int
	StartTime TimeCode
	EndTime   TimeCode
	Text      string
}

func (i SrtItem) String() string {
	return fmt.Sprintf("%d%s%v --> %v%s%s%s", i.Counter, lineEnd, i.StartTime,
		i.EndTime, lineEnd, i.Text, lineEnd)
}

func readLine(r *bufio.Reader) (string, error) {
	str, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}

	str = strings.Trim(str, "\r\n")
	return str, nil
}

func processText(reader *bufio.Reader, item *SrtItem) (string, error) {
	for {
		str, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if str != "" {
			item.Text = item.Text + str + lineEnd
		} else {
			for {
				nextLine, err := readLine(reader)
				if err != nil {
					return "", err
				}

				if nextLine != "" {
					return nextLine, nil
				}
				item.Text = item.Text + str + lineEnd
			}
		}
	}
}

func doReadSrt(reader *bufio.Reader) (*list.List, error) {
	l := list.New()
	counterLine, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	for {
		item := &SrtItem{}

		// process Counter
		item.Counter, err = strconv.Atoi(counterLine)
		if err != nil {
			fmt.Println("parse Counter:", err)
			break
		}

		// process TimeCode
		str, err := readLine(reader)
		if err != nil {
			break
		}
		Str2TimeCode(str[0:12], &item.StartTime)
		Str2TimeCode(str[17:29], &item.EndTime)

		// process Text
		counterLine, err = processText(reader, item)

		if debug {
			fmt.Println("--------------------")
			fmt.Printf("%v", item)
			fmt.Println("--------------------")
		}
		l.PushBack(item)

		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
	}

	return l, nil
}

func ReadSrtFile(filename string) (*list.List, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	return doReadSrt(reader)
}

func doWriteSrt(lists []*list.List, writer *bufio.Writer) error {
	for _, l := range lists {
		for e := l.Front(); e != nil; e = e.Next() {
			fmt.Fprint(writer, e.Value)
		}
	}
	writer.Flush()

	return nil
}

func WriteSrtFile(lists []*list.List, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	return doWriteSrt(lists, writer)
}

func MergeSrt(lists []*list.List, timeOffset []string) {
	len := len(timeOffset)
	lastCounter := lists[0].Len()
	for i := 1; i <= len; i++ {
		for e := lists[i].Front(); e != nil; e = e.Next() {
			item := e.Value.(*SrtItem)
			item.Counter = item.Counter + lastCounter
			item.StartTime.Add(timeOffset[i-1])
			item.EndTime.Add(timeOffset[i-1])
		}
	}
}

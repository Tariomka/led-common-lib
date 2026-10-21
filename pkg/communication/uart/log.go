package uart

import (
	"strconv"
	"strings"
)

// EventKind tells what a cube log line means. Values lists the numbers of the line.
type EventKind int

const (
	EventText            EventKind = iota // any other line
	EventStatus                           // "Status:..." reply, see Event.Status
	EventMode                             // "Mode=<name>": DIP switches changed; Text = mode name
	EventCartoon                          // "Change Cartoon n"; Values[0] = n
	EventNextCartoon                      // "Next Cartoon"
	EventPreviousCartoon                  // "Previous Cartoon"
	EventReplayCartoon                    // "Replay Cartoon"
	EventSpeed                            // "Add speed to", "Sub speed to", "Speed reset to"; Values[0] = speed in 1/100
	EventPlayModel                        // "Play Model=n"; Values[0] = 1 play all, 0 repeat one
	EventPause                            // "DeviceSuppend:n"; Values[0] = 1 paused
	EventDisplayOff                       // "Display off=n"; Values[0] = 1 off
	EventKey                              // "DoKeyN"; Values[0] = N
	EventIR                               // "IR:<hex>"; Values[0] = code
	EventUploadStarted                    // "Show upload started"
	EventUploadOK                         // "Show upload OK"
	EventUploadFailed                     // "Show upload failed"
	EventStartup                          // "Led Cube is starting up": the cube restarted
	EventWatchdogReset                    // "Watchdog reset!"
	EventCubeType                         // "T='<type>'"; Text = type
	EventConfig                           // "Config1=m,Config2=s"; Values = mode number, DIP bits
	EventDevice                           // "DeviceType=t,DeviceNum=n"; Values = 1 master / 0 slave, position
	EventTotal                            // "Total=n"; Values[0] = cubes in the show's display table
	EventRun                              // "Run x=,y=,z=,dir=,type="; Values = x, y, z, dir, type
	EventNoUploadedShow                   // "No uploaded show"
	EventUploadedShow                     // "Uploaded show: n cartoons"; Values[0] = n
	EventNotReady                         // "Not Ready!"
	EventParamError                       // "Param Error!"
	EventNotFound                         // "Not found <kind>:'<name>'"; Text = "<kind>:'<name>'"
	EventWarning                          // "Warn: ..."; Text = the message
	EventError                            // "Save settings error!", "Regedit format error!", "Script error: ..."
)

// Event is a parsed log line.
type Event struct {
	Kind   EventKind
	Line   string
	Text   string
	Values [5]int
	Status Status
}

// Status is the reply to the Status command.
type Status struct {
	Mode       string // Dynamic, Audio, Test or Text
	Cartoon    int    // index of the playing cartoon
	Count      int    // cartoons in the show
	Speed      int    // in 1/100 (100 = 1.00)
	PlayAll    bool   // PlayModel=1: play all cartoons in turn; 0: repeat the current one
	Pause      bool
	DisplayOff bool
	DeviceNum  int // position in the cascade chain, 0 = master
}

// Log lines of the upload, used by the upload client.
const (
	LineUploadStarted = "Show upload started"
	LineUploadOK      = "Show upload OK"
	LineUploadFailed  = "Show upload failed"
)

var exactLines = [...]struct {
	line string
	kind EventKind
}{
	{"Next Cartoon", EventNextCartoon},
	{"Previous Cartoon", EventPreviousCartoon},
	{"Replay Cartoon", EventReplayCartoon},
	{LineUploadStarted, EventUploadStarted},
	{LineUploadOK, EventUploadOK},
	{LineUploadFailed, EventUploadFailed},
	{"Led Cube is starting up", EventStartup},
	{"Watchdog reset!", EventWatchdogReset},
	{"No uploaded show", EventNoUploadedShow},
	{"Not Ready!", EventNotReady},
	{"Param Error!", EventParamError},
	{"Save settings error!", EventError},
	{"Regedit format error!", EventError},
}

// ParseLine turns a known log line into an event; other lines are EventText.
func ParseLine(line string) Event {
	event := Event{Kind: EventText, Line: line}
	for _, exact := range exactLines {
		if line == exact.line {
			event.Kind = exact.kind
			if event.Kind == EventError {
				event.Text = line
			}
			return event
		}
	}

	ok := false
	if rest, found := cut(line, "Status:"); found {
		event.Kind, ok = EventStatus, parseStatus(rest, &event.Status)
	} else if rest, found := cut(line, "Mode="); found {
		event.Kind, event.Text, ok = EventMode, rest, true
	} else if rest, found := cut(line, "Change Cartoon "); found {
		event.Kind, ok = EventCartoon, number(rest, &event.Values[0])
	} else if rest, found := speedLine(line); found {
		event.Kind, ok = EventSpeed, hundredths(rest, &event.Values[0])
	} else if rest, found := cut(line, "Play Model="); found {
		event.Kind, ok = EventPlayModel, number(rest, &event.Values[0])
	} else if rest, found := cut(line, "DeviceSuppend:"); found {
		event.Kind, ok = EventPause, number(rest, &event.Values[0])
	} else if rest, found := cut(line, "Display off="); found {
		event.Kind, ok = EventDisplayOff, number(rest, &event.Values[0])
	} else if rest, found := cut(line, "DoKey"); found {
		event.Kind, ok = EventKey, number(rest, &event.Values[0])
	} else if rest, found := cut(line, "IR:"); found {
		code, err := strconv.ParseUint(rest, 16, 32)
		event.Kind, event.Values[0], ok = EventIR, int(code), err == nil
	} else if rest, found := cut(line, "T='"); found && strings.HasSuffix(rest, "'") {
		event.Kind, event.Text, ok = EventCubeType, rest[:len(rest)-1], true
	} else if strings.HasPrefix(line, "Config1=") {
		event.Kind, ok = EventConfig, values(line, event.Values[:], "Config1", "Config2")
	} else if strings.HasPrefix(line, "DeviceType=") {
		event.Kind, ok = EventDevice, values(line, event.Values[:], "DeviceType", "DeviceNum")
	} else if strings.HasPrefix(line, "Total=") {
		event.Kind, ok = EventTotal, values(line, event.Values[:], "Total")
	} else if rest, found := cut(line, "Run "); found {
		event.Kind, ok = EventRun, values(rest, event.Values[:], "x", "y", "z", "dir", "type")
	} else if rest, found := cut(line, "Uploaded show: "); found && strings.HasSuffix(rest, " cartoons") {
		event.Kind, ok = EventUploadedShow, number(rest[:len(rest)-len(" cartoons")], &event.Values[0])
	} else if rest, found := cut(line, "Not found "); found {
		event.Kind, event.Text, ok = EventNotFound, rest, true
	} else if rest, found := cut(line, "Warn: "); found {
		event.Kind, event.Text, ok = EventWarning, rest, true
	} else if strings.HasPrefix(line, "Script error: ") {
		event.Kind, ok = EventError, true
	}

	if !ok {
		return Event{Kind: EventText, Line: line}
	}
	if event.Kind == EventError {
		event.Text = line
	}
	return event
}

func cut(line, prefix string) (string, bool) {
	if strings.HasPrefix(line, prefix) {
		return line[len(prefix):], true
	}
	return "", false
}

func speedLine(line string) (string, bool) {
	for _, prefix := range [...]string{"Add speed to ", "Sub speed to ", "Speed reset to "} {
		if rest, found := cut(line, prefix); found {
			return rest, true
		}
	}
	return "", false
}

func number(text string, value *int) bool {
	n, err := strconv.Atoi(text)
	*value = n
	return err == nil
}

// hundredths parses a speed such as "1.50" as 150.
func hundredths(text string, value *int) bool {
	whole, fraction, _ := strings.Cut(text, ".")
	if len(fraction) > 2 {
		return false
	}
	for len(fraction) < 2 {
		fraction += "0"
	}

	var units, cents int
	if !number(whole, &units) || !number(fraction, &cents) || units < 0 || cents < 0 {
		return false
	}
	*value = units*100 + cents
	return true
}

// values reads "key1=v1,key2=v2,..." with exactly the given keys, in order.
func values(text string, values []int, keys ...string) bool {
	for i, key := range keys {
		rest, found := cut(text, key)
		if !found || len(rest) == 0 || rest[0] != '=' {
			return false
		}

		value, next, more := strings.Cut(rest[1:], ",")
		if more != (i < len(keys)-1) || !number(value, &values[i]) {
			return false
		}
		text = next
	}
	return true
}

func parseStatus(text string, status *Status) bool {
	for len(text) > 0 {
		var field string
		field, text, _ = strings.Cut(text, ",")
		key, value, found := strings.Cut(field, "=")
		if !found {
			return false
		}

		var n int
		ok := true
		switch key {
		case "Mode":
			status.Mode = value
		case "Cartoon":
			ok = number(value, &status.Cartoon)
		case "Count":
			ok = number(value, &status.Count)
		case "Speed":
			ok = hundredths(value, &status.Speed)
		case "PlayModel":
			ok = number(value, &n)
			status.PlayAll = n != 0
		case "Pause":
			ok = number(value, &n)
			status.Pause = n != 0
		case "DisplayOff":
			ok = number(value, &n)
			status.DisplayOff = n != 0
		case "DeviceNum":
			ok = number(value, &status.DeviceNum)
		}
		if !ok {
			return false
		}
	}
	return true
}

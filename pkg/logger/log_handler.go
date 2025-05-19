package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	reset = "\033[0m"

	bold      = 1
	italic    = 3
	underline = 4

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97

	timeFormat = "[15:04:05.000]"
)

type LogHandler struct {
	handler slog.Handler
	print   func(message string)

	mutex  *sync.Mutex
	buffer *bytes.Buffer
}

func NewLogHandler(printCallback func(message string), opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if printCallback == nil {
		printCallback = func(message string) { fmt.Println(message) }
	}

	buffer := &bytes.Buffer{}
	return &LogHandler{
		handler: slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		print:  printCallback,
		mutex:  &sync.Mutex{},
		buffer: buffer,
	}
}

func (this *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	payload, err := this.getPayload(ctx, record)
	if err != nil {
		return err
	}

	this.print(payload)
	return nil
}

func (this *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return this.handler.Enabled(ctx, level)
}

func (this *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{
		handler: this.handler.WithAttrs(attrs),
		print:   this.print,
		mutex:   this.mutex,
		buffer:  this.buffer,
	}
}

func (this *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{
		handler: this.handler.WithGroup(name),
		print:   this.print,
		mutex:   this.mutex,
		buffer:  this.buffer,
	}
}

func (this *LogHandler) getPayload(ctx context.Context, record slog.Record) (string, error) {
	source, attributes, err := this.handleAttributes(ctx, record)
	if err != nil {
		return "", err
	}

	bytes, err := json.MarshalIndent(attributes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error when marshaling attrs: %w", err)
	}

	components := []string{}
	components = append(components, colorize(lightGray, record.Time.Format(timeFormat)))
	if source != "" {
		components = append(components, source)
	}
	components = append(
		components,
		this.handleLevel(record),
		colorize(white, record.Message),
		colorize(darkGray, string(bytes)))

	payload := strings.Join(components, " ")
	return payload, nil
}

func (this *LogHandler) handleLevel(record slog.Record) string {
	level := record.Level.String() + ":"

	switch record.Level {
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	return level
}

func (this *LogHandler) handleAttributes(
	ctx context.Context,
	record slog.Record) (source string, attributes map[string]any, err error) {
	this.mutex.Lock()
	defer func() {
		this.buffer.Reset()
		this.mutex.Unlock()
	}()

	if err := this.handler.Handle(ctx, record); err != nil {
		return "", nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	if err := json.Unmarshal(this.buffer.Bytes(), &attributes); err != nil {
		return "", nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}

	return this.handleSource(attributes), attributes, nil
}

func (this *LogHandler) handleSource(attributes map[string]any) string {
	attribute, ok := attributes["source"]
	if !ok {
		return ""
	}

	delete(attributes, "source")
	switch value := attribute.(type) {
	case string:
		return stylize(bold, colorize(green, fmt.Sprintf("[%s]", value)))

	case map[string]any:
		fileAttribute, getOk := value["file"]
		file, castOk := fileAttribute.(string)
		if !getOk || !castOk {
			return ""
		}

		filename := filepath.Base(file)
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		source := strings.ToUpper(name)

		lineAttribute, getOk := value["line"]
		line, castOk := lineAttribute.(float64)
		if !getOk || !castOk {
			return stylize(bold, colorize(green, fmt.Sprintf("[%s]", source)))
		}

		return stylize(bold, colorize(green, fmt.Sprintf("[%s(%.0f)]", source, line)))

	default:
		fmt.Printf("[LOG_HANDLER] received source with unknown structure: %#v\n", value)
		return ""
	}
}

func suppressDefaults(next func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey || attr.Key == slog.LevelKey || attr.Key == slog.MessageKey {
			return slog.Attr{}
		}

		if next == nil {
			return attr
		}

		return next(groups, attr)
	}
}

func colorize(colorCode int, value string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), value, reset)
}

func stylize(style int, value string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(style), value, reset)
}

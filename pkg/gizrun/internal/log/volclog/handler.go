package volclog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volc-sdk-golang/service/tls/common"
	"github.com/volcengine/volc-sdk-golang/service/tls/pb"
	"github.com/volcengine/volc-sdk-golang/service/tls/producer"
)

type Producer interface {
	SendLog(shardHash, topic, source, filename string, log *pb.Log, callBack producer.CallBack) error
	Start()
	Close()
	ForceClose()
}

type Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
	APIKey          string

	TopicID   string
	Source    string
	FileName  string
	ShardHash string

	Level            slog.Leveler
	AddSource        bool
	EnableNanosecond bool

	ProducerConfig func(*producer.Config)
}

type Handler struct {
	producer Producer
	config   Config
	level    slog.Leveler
	attrs    []field
	groups   []string
}

type field struct {
	key   string
	value string
}

func NewHandler(config Config) (*Handler, error) {
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errors.New("volclog: endpoint is required")
	}
	if strings.TrimSpace(config.Region) == "" {
		return nil, errors.New("volclog: region is required")
	}
	if strings.TrimSpace(config.TopicID) == "" {
		return nil, errors.New("volclog: topic id is required")
	}
	producerConfig := producer.GetDefaultProducerConfig()
	producerConfig.ClientConfig = common.ClientConfig{
		Endpoint:        config.Endpoint,
		AccessKeyID:     config.AccessKeyID,
		AccessKeySecret: config.AccessKeySecret,
		SecurityToken:   config.SecurityToken,
		APIKey:          config.APIKey,
		Region:          config.Region,
	}
	producerConfig.EnableNanosecond = config.EnableNanosecond
	if config.ProducerConfig != nil {
		config.ProducerConfig(producerConfig)
	}
	return NewHandlerWithProducer(config, producer.NewProducer(producerConfig)), nil
}

func NewHandlerWithProducer(config Config, p Producer) *Handler {
	if config.Source == "" {
		config.Source = "gizclaw"
	}
	if config.FileName == "" {
		config.FileName = "slog"
	}
	h := &Handler{producer: p, config: config, level: config.Level}
	if h.level == nil {
		h.level = slog.LevelInfo
	}
	if h.producer != nil {
		h.producer.Start()
	}
	return h
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	if h == nil {
		return false
	}
	min := slog.LevelInfo
	if h.level != nil {
		min = h.level.Level()
	}
	return level >= min
}

func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	if h == nil || h.producer == nil {
		return nil
	}
	contents := []field{
		{key: "level", value: record.Level.String()},
		{key: "msg", value: record.Message},
	}
	if h.config.AddSource && record.PC != 0 {
		contents = append(contents, sourceField(record.PC))
	}
	contents = append(contents, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		contents = appendAttr(contents, h.groups, attr)
		return true
	})

	var logTime int64
	if !record.Time.IsZero() {
		logTime = record.Time.UnixMilli()
	}
	log := &pb.Log{Time: logTime, Contents: make([]*pb.LogContent, 0, len(contents))}
	if h.config.EnableNanosecond && !record.Time.IsZero() {
		log.OptionalTimeNs = &pb.Log_TimeNs{TimeNs: uint32(record.Time.Nanosecond() % int(time.Millisecond))}
	}
	for _, item := range contents {
		if item.key == "" {
			continue
		}
		log.Contents = append(log.Contents, &pb.LogContent{Key: item.key, Value: item.value})
	}
	return h.producer.SendLog(h.config.ShardHash, h.config.TopicID, h.config.Source, h.config.FileName, log, nil)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil {
		return (*Handler)(nil)
	}
	next := h.clone()
	for _, attr := range attrs {
		next.attrs = appendAttr(next.attrs, h.groups, attr)
	}
	return next
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if h == nil {
		return (*Handler)(nil)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(append([]string(nil), h.groups...), name)
	return next
}

func (h *Handler) Close() error {
	if h == nil || h.producer == nil {
		return nil
	}
	h.producer.Close()
	return nil
}

func (h *Handler) ForceClose() {
	if h == nil || h.producer == nil {
		return
	}
	h.producer.ForceClose()
}

func (h *Handler) clone() *Handler {
	next := *h
	next.attrs = append([]field(nil), h.attrs...)
	next.groups = append([]string(nil), h.groups...)
	return &next
}

func appendAttr(out []field, groups []string, attr slog.Attr) []field {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return out
	}
	if attr.Value.Kind() == slog.KindGroup {
		name := strings.TrimSpace(attr.Key)
		nextGroups := groups
		if name != "" {
			nextGroups = append(append([]string(nil), groups...), name)
		}
		for _, child := range attr.Value.Group() {
			out = appendAttr(out, nextGroups, child)
		}
		return out
	}
	key := joinKey(groups, attr.Key)
	if key == "" {
		return out
	}
	return append(out, field{key: key, value: valueString(attr.Value)})
}

func joinKey(groups []string, key string) string {
	parts := make([]string, 0, len(groups)+1)
	for _, group := range groups {
		if group = strings.TrimSpace(group); group != "" {
			parts = append(parts, group)
		}
	}
	if key = strings.TrimSpace(key); key != "" {
		parts = append(parts, key)
	}
	return strings.Join(parts, ".")
}

func valueString(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		return strconv.FormatBool(value.Bool())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'g', -1, 64)
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindLogValuer:
		return valueString(value.Resolve())
	case slog.KindAny:
		any := value.Any()
		if err, ok := any.(error); ok {
			return err.Error()
		}
		data, err := json.Marshal(any)
		if err == nil {
			return string(data)
		}
		return fmt.Sprint(any)
	default:
		return value.String()
	}
}

func sourceField(pc uintptr) field {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if frame.File == "" {
		return field{}
	}
	return field{key: "source", value: frame.File + ":" + strconv.Itoa(frame.Line)}
}

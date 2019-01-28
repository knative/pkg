package cloudevents

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CloudEventEncoding is used to tell the builder which encoding to select.
// the default is Binary.
type CloudEventEncoding int

const (
	// Binary
	BinaryEncodingV01 CloudEventEncoding = 0
	// Structured
	StructuredEncodingV01 CloudEventEncoding = 1
)

// Builder holds settings that do not change over CloudEvents. It is intended
// to represent a builder of only a single CloudEvent type.
type Builder struct {
	Source    string
	EventType string
	Encoding  CloudEventEncoding
}

// Build produces a http request with the constant data embedded in the builder
// merged with the new data provided in the build function. The request will
// send a pre-assembled cloud event to the given target. The target is assumed
// to be a URL with a scheme, ie: "http://localhost:8080"
func (b *Builder) Build(target string, data interface{}) (*http.Request, error) {
	if b.Source == "" {
		return nil, fmt.Errorf("Build.Source is empty")
	}
	if b.EventType == "" {
		return nil, fmt.Errorf("Build.EventType is empty")
	}

	ctx := b.cloudEventsContext()

	switch b.Encoding {
	case BinaryEncodingV01:
		return Binary.NewRequest(target, data, ctx)
	case StructuredEncodingV01:
		return Structured.NewRequest(target, data, ctx)
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", b.Encoding)
	}
}

// cloudEventsContext creates a CloudEvent context object, assumes applicaiton/json as the content
// type.
func (b *Builder) cloudEventsContext() EventContext {
	return EventContext{
		CloudEventsVersion: CloudEventsVersion,
		EventType:          b.EventType,
		EventID:            uuid.New().String(),
		Source:             b.Source,
		ContentType:        "application/json",
		EventTime:          time.Now(),
	}
}

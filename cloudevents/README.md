# Knative CloudEvents SDK

This library produces CloudEvents in version 0.1 compatible form. To learn more
about CloudEvents, see the [Specification](https://github.com/cloudevents/spec).

There are two roles the SDK fulfills: the [producer](#producer) and the
[consumer](#consumer). The producer creates a cloud event in either
[Binary](#binary) or [Structured](#structured) request format. The producer
assembles and sends the event through an HTTP endpoint. The consumer will
inspect the incoming HTTP request and select the correct decode format.

This SDK should be wire-compatible with any other producer or consumer of the
supported versions of CloudEvents.

## Getting Started

CloudEvents acts as the envelope in which to send a custom object. Create a
CloudEvent type for the events you will be producing.

Example CloudEvent Type: `dev.knative.cloudevent.example`

Select a source to identify the originator of this CloudEvent. It should be a
valid URI.

Example CloudEvent Source: `https://github.com/knative/pkg#cloudevents-example`

And finally, create a struct that will be the data inside the CloudEvent,
example:

```go
type Example struct {
    Sequence int    `json:"id"`
    Message    string `json:"message"`
}
```

### Producer

The producer will create a new `Example` object, fill out the CloudEvent struct,
and post the event via a [Binary](#binary) or [Structured](#structured) request
format.

```go

package main

import (
    "github.com/google/uuid"
    "github.com/knative/pkg/cloudevents"
    "io/ioutil"
    "log"
    "net/http"
    "time"
)

type Example struct {
    Sequence int    `json:"id"`
    Message  string `json:"message"`
}

func main() {
    target := "http://localhost:8080"
    eventType := "dev.knative.cloudevent.example"
    eventSource := "https://github.com/knative/pkg#cloudevents-example"
    data := &Example{
        Sequence: 0,
        Message:  "hello, world!",
    }
    ctx := cloudevents.EventContext{
        CloudEventsVersion: cloudevents.CloudEventsVersion,
        EventType:          eventType,
        EventID:            uuid.New().String(),
        Source:             eventSource,
        EventTime:          time.Now(),
    }
    req, err := cloudevents.Binary.NewRequest(target, data, ctx)
    if err != nil {
        log.Printf("failed to create http request: %s", err)
        return
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("failed to do POST: %v", err)
        return
    }
    defer resp.Body.Close()
    log.Printf("response Status: %s", resp.Status)
    body, _ := ioutil.ReadAll(resp.Body)
    log.Printf("response Body: %s", string(body))
}

```

### Consumer

The consumer will listen for a post and then inspect the headers to understand
how to decode the request.

```go

package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/knative/pkg/cloudevents"
)

type Example struct {
    Sequence int    `json:"id"`
    Message  string `json:"message"`
}

func handler(ctx context.Context, data *Example) {
    metadata := cloudevents.FromContext(ctx)
    log.Printf("[%s] %s %s: %d,%q", metadata.EventTime.Format(time.RFC3339), metadata.ContentType, metadata.Source, data.Sequence, data.Message)
}

func main() {
    log.Print("ready and listening on port 8080")
    log.Fatal(http.ListenAndServe(":8080", cloudevents.Handler(handler)))
}


```

## Request Formats

### Binary

Changes to the producer code required to leverage binary request format:

```go
req, err := cloudevents.Binary.NewRequest(target, data, ctx)

```

### Structured

Changes to the producer code to leverage structured request format:

```go
req, err := cloudevents.Structured.NewRequest(target, data, ctx)
```

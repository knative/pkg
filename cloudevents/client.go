package cloudevents

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Client wraps Builder, and is intended to be configured for a single event
// type and target
type Client struct {
	Builder
	Target string

	send chan interface{}
	done chan bool
}

const (
	// MAX_SEND_CHANNEL is the max buffered channel length for the send channel
	MAX_SEND_CHANNEL = 10
)

// NewClient returns a CloudEvent Client used to send CloudEvents. It is
// intended that a user would create a new client for each tuple of eventType
// and target. This is a n optional helper method to avoid the tricky creation
// of the embedded Builder struct.
func NewClient(eventType, source, target string) *Client {
	c := &Client{
		Builder: Builder{
			Source:    source,
			EventType: eventType,
		},
		Target: target,
	}
	return c
}

// Send creates a request based on the client's settings and sends the data
// struct to the target set for this client. It returns error if there was an
// issue sending the event, otherwise nil means the event was accepted.
func (c *Client) Send(data interface{}) error {
	resp, err := c.RequestSend(data)
	if err != nil {
		return err
	}
	if accepted(resp) {
		return nil
	}
	return fmt.Errorf("error sending cloudevent: %s", status(resp))
}

// RequestSend uses the internal builder to make a request with the provided
// data struct using the previously set parameters of event type and target.
// Use this instead of client.Send() if processing of the http response directly
// is required.
func (c *Client) RequestSend(data interface{}) (*http.Response, error) {
	req, err := c.Build(c.Target, data)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}

// Channel returns a channel that can be used to invoke Client.Send via a chan.
// This method has a side effect of another thread to monitor the send channel.
// Call client.Close() to shutdown the monitor thread.
// Experimental, error handling is not fully developed.
func (c *Client) Channel() chan<- interface{} {
	if c.send == nil {
		c.done = make(chan bool)
		c.send = make(chan interface{}, MAX_SEND_CHANNEL)
		go c.monitorSend()
	}
	return c.send
}

// Close ends the channel monitor produced by calling Channel()
// Experimental, error handling is not fully developed.
func (c *Client) Close() {
	if c.send == nil {
		return
	}
	c.done <- true
	close(c.send)
	c.send = nil
}

// monitorSend is the thread that will watch the send channel and call
// client.Sent() with the provided data struct. It will exit if the send channel
// closes or something is received on done channel.
func (c *Client) monitorSend() {
	for {
		select {
		case data, ok := <-c.send:
			if ok == false {
				break
			}
			if err := c.Send(data); err != nil {
				log.Printf("error sending: %v", err)
			}
		case <-c.done:
			return
		}
	}
}

// accept is a helper method to understand if the respose from the target
// accepted the CloudEvent.
func accepted(resp *http.Response) bool {
	if resp.StatusCode == 204 {
		return true
	}
	return false
}

// status is a helper method to read the response of the target.
func status(resp *http.Response) string {
	if accepted(resp) {
		return "sent"
	}
	status := resp.Status
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Status[%s] error reading response body: %v", status, err)
	}
	return fmt.Sprintf("Status[%s] %s", status, body)
}

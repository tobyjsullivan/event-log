package writer

import (
    "net/url"
    "encoding/base64"
    "net/http"
    "fmt"
    "github.com/tobyjsullivan/ues-sdk/event"
    "errors"
)

type EventWriter struct {
    svcUrl *url.URL
}

type EventWriterConfig struct {
    ServiceUrl string
}

func New(conf *EventWriterConfig) (*EventWriter, error) {
    svcUrl, err := url.Parse(conf.ServiceUrl)
    if err != nil {
        return nil, err
    }

    return &EventWriter{
        svcUrl: svcUrl,
    }, nil
}

func (s *EventWriter) PutEvent(e *event.Event) error {
    previousId :=  e.PreviousEvent.String()

    b64Data := base64.StdEncoding.EncodeToString(e.Data)

    eventsEndpoint := s.svcUrl.ResolveReference(&url.URL{Path:"./events"})
    resp, err := http.PostForm(eventsEndpoint.String(), url.Values{
        "previous": {previousId},
        "type": {e.Type},
        "data": {b64Data},
    })

    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        return errors.New(fmt.Sprintf("Unexpected status from event-store. %s", resp.Status))
    }

    return nil
}


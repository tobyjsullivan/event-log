package store

import (
    "github.com/tobyjsullivan/event-store.v3/events"
    "net/http"
    "net/url"
    "encoding/base64"
    "errors"
    "fmt"
)

type Store struct {
    svcUrl *url.URL
}

type StoreConfig struct {
    EventStoreServiceUrl string
}

func New(conf *StoreConfig) (*Store, error) {
    svcUrl, err := url.Parse(conf.EventStoreServiceUrl)
    if err != nil {
        return nil, err
    }

    return &Store{
        svcUrl: svcUrl,
    }, nil
}

func (s *Store) WriteEvent(e *events.Event) error {
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

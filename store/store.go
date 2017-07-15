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

}

type StoreConfig struct {

}

func New(conf *StoreConfig) *Store {
    return &Store{}
}

func (s *Store) WriteEvent(e *events.Event) error {
    previousId :=  e.PreviousEvent.String()

    b64Data := base64.StdEncoding.EncodeToString(e.Data)

    resp, err := http.PostForm("http://event-store:3000/events", url.Values{
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

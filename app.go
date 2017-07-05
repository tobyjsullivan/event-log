package main

import (
    "os"

    "fmt"
    "net/http"

    "github.com/urfave/negroni"
    "github.com/gorilla/mux"

    _ "github.com/lib/pq"
    "database/sql"
    "log"
    "github.com/tobyjsullivan/event-log/eventlog"
    "github.com/tobyjsullivan/event-store.v3/events"
    "encoding/base64"
    "errors"
    "encoding/hex"
)

var (
    logger *log.Logger
    db *sql.DB
)

func init() {
    logger = log.New(os.Stdout, "[svc] ", 0)

    pgHostname := os.Getenv("PG_HOSTNAME")
    pgUsername := os.Getenv("PG_USERNAME")
    pgPassword := os.Getenv("PG_PASSWORD")
    pgDatabase := os.Getenv("PG_DATABASE")

    dbConnOpts := fmt.Sprintf("host='%s' user='%s' dbname='%s' password='%s' sslmode=disable",
        pgHostname, pgUsername, pgDatabase, pgPassword)

    logger.Println("Connecting to DB...")
    var err error
    db, err = sql.Open("postgres", dbConnOpts)
    if err != nil {
        panic(err.Error())
    }
}

func main() {
    r := buildRoutes()

    n := negroni.New()
    n.UseHandler(r)

    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    n.Run(":" + port)
}

func buildRoutes() http.Handler {
    r := mux.NewRouter()
    r.HandleFunc("/", statusHandler).Methods("GET")
    r.HandleFunc("/commands/create-log", createLogHandler).Methods("POST")
    r.HandleFunc("/commands/append-event", appendEventHandler).Methods("POST")

    return r
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "The service is online!\n")
}

func createLogHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    logId := r.Form.Get("log-id")
    if logId == "" {
        http.Error(w, "log-id must be set.", http.StatusBadRequest)
        return
    }

    var id eventlog.LogID
    err = id.Parse(logId)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    err = createLog(db, id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.Write([]byte{})
}

func createLog(conn *sql.DB, logId eventlog.LogID) error {
    bHead := [32]byte{}

    res, err := conn.Exec(`INSERT INTO logs(ext_lookup_key, head) VALUES ($1, $2)`, logId[:], bHead[:])
    if err != nil {
        return err
    }

    numRows, err := res.RowsAffected()
    if err != nil {
        return err
    }
    logger.Println("Rows affected:", numRows)

    return nil
}

func appendEventHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    paramLogId := r.Form.Get("log-id")
    if paramLogId == "" {
        http.Error(w, "log-id must be set.", http.StatusBadRequest)
        return
    }

    eventType := r.Form.Get("event-type")
    if eventType == "" {
        http.Error(w, "event-type must be set.", http.StatusBadRequest)
        return
    }

    eventData := r.Form.Get("event-data")
    if eventData == "" {
        http.Error(w, "event-data must be set.", http.StatusBadRequest)
        return
    }

    var logId eventlog.LogID
    err = logId.Parse(paramLogId)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var parsedData []byte
    parsedData, err = base64.StdEncoding.DecodeString(eventData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get the current log head
    headId, err := getLogHead(db, logId)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send the event to event-store service
    e := &events.Event{
        PreviousEvent: headId,
        Type: eventType,
        Data: parsedData,
    }
    newEventId, err := createEvent(e)

    // TODO Update the log head
    err = updateLogHead(db, logId, headId, newEventId)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprint(w, "Updated log: ", hex.EncodeToString(newEventId[:]))
}

func getLogHead(conn *sql.DB, id eventlog.LogID) (events.EventID, error) {
    var head []byte
    err := conn.QueryRow(`SELECT head FROM logs WHERE ext_lookup_key=$1`, id[:]).Scan(&head)
    if err != nil {
        return events.EventID{}, err
    }

    var out events.EventID
    copy(out[:], head)
    return out, nil
}

func updateLogHead(conn *sql.DB, logId eventlog.LogID, expectedHead events.EventID, newHead events.EventID) error {
    res, err := conn.Exec(`UPDATE logs SET head=$1 WHERE ext_lookup_key=$2 AND head=$3`, newHead[:], logId[:], expectedHead[:])
    if err != nil {
        return err
    }

    rowsAffected, err := res.RowsAffected()
    if err != nil {
        return err
    }

    if rowsAffected != 1 {
        return errors.New("There was no log with matching head or id.")
    }

    return nil
}

func createEvent(e *events.Event) (events.EventID, error) {
    // TODO Send the event to event-store service
    return e.ID(), nil
}

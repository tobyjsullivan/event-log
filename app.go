package main

import (
    "os"

    "fmt"
    "net/http"

    "github.com/gorilla/mux"
    "github.com/urfave/negroni"

    "database/sql"
    "encoding/base64"
    "encoding/hex"
    "errors"
    "log"

    _ "github.com/lib/pq"
    eventLog "github.com/tobyjsullivan/event-log/log"
    "github.com/tobyjsullivan/ues-sdk/event/writer"
    "github.com/tobyjsullivan/ues-sdk/event"
)

var (
    logger     *log.Logger
    db         *sql.DB
    eventWriter *writer.EventWriter
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
        logger.Println("Error initializing connection to Postgres DB.", err.Error())
        panic(err.Error())
    }

    eventWriter, err = writer.New(&writer.EventWriterConfig{
        ServiceUrl: os.Getenv("EVENT_STORE_API"),
    })
    if err != nil {
        logger.Println("Error initializing connection to event store.", err.Error())
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

// Placeholder for backward-compatibility
// TODO Remove on next major version release
func createLogHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        logger.Println("Error parsing form.", err.Error())
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    logId := r.Form.Get("log-id")
    if logId == "" {
        logger.Println("Error getting log-id from form.", err.Error())
        http.Error(w, "log-id must be set.", http.StatusBadRequest)
        return
    }

    // NO-OP
    // We don't need to create the log until it's written to.

    w.Write([]byte(fmt.Sprintf("Log created: %s", logId)))
}

func createLogIfNotExists(conn *sql.DB, logId eventLog.LogID) error {
    bHead := [32]byte{}

    res, err := conn.Exec(`INSERT INTO logs(ext_lookup_key, head) VALUES ($1, $2) ON CONFLICT DO NOTHING`, logId[:], bHead[:])
    if err != nil {
        logger.Println("Error inserting new log record.", err.Error())
        return err
    }

    numRows, err := res.RowsAffected()
    if err != nil {
        logger.Println("Error reading RowsAffected.", err.Error())
        return err
    }
    logger.Println("Rows affected:", numRows)

    return nil
}

func appendEventHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        logger.Println("Error parsing form during append-event.", err.Error())
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    paramLogId := r.Form.Get("log-id")
    if paramLogId == "" {
        logger.Println("Error parsing log-id.", err.Error())
        http.Error(w, "log-id must be set.", http.StatusBadRequest)
        return
    }

    eventType := r.Form.Get("event-type")
    if eventType == "" {
        logger.Println("Error parsing event-tyoe.", err.Error())
        http.Error(w, "event-type must be set.", http.StatusBadRequest)
        return
    }

    eventData := r.Form.Get("event-data")
    if eventData == "" {
        logger.Println("Error parsing event-data.", err.Error())
        http.Error(w, "event-data must be set.", http.StatusBadRequest)
        return
    }

    var logId eventLog.LogID
    err = logId.Parse(paramLogId)
    if err != nil {
        logger.Println("Error parsing LogID during event append.", err.Error())
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var parsedData []byte
    parsedData, err = base64.StdEncoding.DecodeString(eventData)
    if err != nil {
        logger.Println("Error parsing data string.", err.Error())
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get the current log head
    headId, err := getLogHead(db, logId)
    if err != nil {
        logger.Println("Error reading current log head.", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    zero := event.EventID{}
    if headId == zero {
        createLogIfNotExists(db, logId)
    }

    // Send the event to event-store service
    e := &event.Event{
        PreviousEvent: headId,
        Type:          eventType,
        Data:          parsedData,
    }
    newEventId, err := createEvent(e)

    // Update the log head
    err = updateLogHead(db, logId, headId, newEventId)
    if err != nil {
        logger.Println("Error updating log head.", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprint(w, "Updated log: ", hex.EncodeToString(newEventId[:]))
}

func getLogHead(conn *sql.DB, id eventLog.LogID) (event.EventID, error) {
    var head []byte
    err := conn.QueryRow(`SELECT head FROM logs WHERE ext_lookup_key=$1`, id[:]).Scan(&head)
    if err == sql.ErrNoRows {
        // Treat a missing log as an empty log
        return event.EventID{}, nil
    }
    if err != nil {
        logger.Println("Error executing SELECT for log head lookup.", err.Error())
        return event.EventID{}, err
    }

    var out event.EventID
    copy(out[:], head)
    return out, nil
}

func updateLogHead(conn *sql.DB, logId eventLog.LogID, expectedHead event.EventID, newHead event.EventID) error {
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

func createEvent(e *event.Event) (event.EventID, error) {
    err := eventWriter.PutEvent(e)
    if err != nil {
        logger.Println("Error writing event.", err.Error())
        return event.EventID{}, err
    }

    return e.ID(), nil
}

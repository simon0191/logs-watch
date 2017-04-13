package main

import (
	"bytes"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type App struct {
	drainToken string
	router     *mux.Router
	alerts     []string
}

func NewApp(drainToken string, alerts []string) *App {
	app := App{
		drainToken: drainToken,
		router:     mux.NewRouter(),
		alerts:     alerts,
	}

	app.ConfigureRoutes()

	return &app
}

func (app *App) ConfigureRoutes() {
	app.router.HandleFunc("/", app.handleLog).Methods("POST")
}

func (app *App) handleLog(w http.ResponseWriter, r *http.Request) {
	/*
		Logplex-Msg-Count: The number of messages encoded in the body of this request. 2 in the example above. You can use this field as a sanity check to detect if you have not parsed the body correctly.
		Logplex-Frame-Id: The unique identifier for this request. If this request is retried for some reason (non-2xx response code, network connection failure, etc.) this identifier will allow you to spot duplicate requests.
		Logplex-Drain-Token: This is the unique identifier for the Logplex drain. It will be the same identifier you see if you run heroku drains for your app.
	*/

	if token := r.Header.Get("Logplex-Drain-Token"); token != app.drainToken {
		log.Printf("Invalid Token: %s", token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//TODO: search in Redis
	if frameId := r.Header.Get("Logplex-Frame-Id"); app.existFrameId(frameId) {
		log.Printf("Repeated Logplex-Frame-Id: %s", frameId)
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}
	lines := bytes.Split(body, []byte("\n"))

	for _, line := range lines {
		for _, alert := range app.alerts {
			if bytes.Contains(line, []byte(alert)) {
				log.Printf("!!! %s", string(line))
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (app *App) existFrameId(frameId string) bool {
	//TODO: check in the DB
	return false
}

func main() {
	app := NewApp(
		os.Getenv("DRAIN_TOKEN"),
		[]string{"Completed 500 Internal Server Error"},
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Println(http.ListenAndServe(":"+port, app.router))
}

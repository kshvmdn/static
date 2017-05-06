package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/patrickmn/go-cache"
	"github.com/stacktic/dropbox"
)

type (
	service interface {
		route() *route
		setRoute(*route)

		payload() *payload
		setPayload(*payload)

		getFile(string) *file
		setFile(string, *file)

		fetchFile(string) error
		write(http.ResponseWriter, *http.Request, string)

		poll(string) error
	}

	route struct {
		Path     string `json:"path" yaml:"path"`
		Endpoint string `json:"endpoint" yaml:"endpoint"`
	}

	file struct {
		name        string
		data        []byte
		entry       *dropbox.Entry
		notFound    bool
		lastFetched time.Time
	}

	payload struct {
		lastModified time.Time
	}
)

var (
	c      = cache.New(-1, time.Hour*24)
	db     *dropbox.Dropbox
	routes []route

	configPtr   = flag.String("config", "", "Optional config. file for multiple paths & endpoints.")
	endpointPtr = flag.String("endpoint", "/", "Endpoint to serve content at.")
	pathPtr     = flag.String("path", "", "Dropbox path to serve content from.")
	portPtr     = flag.Int("port", 3030, "Server port.")
	refreshPtr  = flag.Bool("refresh-on-dir-change", true, "Refresh on directory change. The alternate (when false) is to only refresh a file when the file itself changes.")
	servicePtr  = flag.String("service", "", "Service to be used (\"dropbox\" or \"drive\").")
	sleepPtr    = flag.Int("sleep", 0, "Time to wait between polls (in seconds).")
)

func readConfig(configFile string, dest *[]route) error {
	raw, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}

	switch filepath.Ext(configFile) {
	case ".json":
		json.Unmarshal(raw, &dest)
	case ".yml", ".yaml":
		yaml.Unmarshal(raw, &dest)
	}

	return nil
}

func startPoll(s service) {
	for {
		if err := s.poll(s.route().Path); err != nil {
			log.Println(err)
			time.Sleep(time.Minute)
		}
	}
}

func makeHandler(s service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := filepath.Join(s.route().Path, r.URL.Path)

		if s.getFile(key) == nil {
			s.setFile(key, &file{})
		}

		v, found := c.Get(key)
		if found {
			s.setRoute(v.(*route))
			if !s.getFile(key).lastFetched.Before(s.payload().lastModified) {
				s.write(w, r, key)
				return
			}
		}

		if err := s.fetchFile(key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.write(w, r, key)
	}
}

func main() {
	flag.Parse()
	routes = make([]route, 0)

	if *servicePtr == "" {
		fmt.Println("Expected -service flag (run static -help).")
		os.Exit(1)
	}

	if *configPtr != "" {
		// Read contents from file at configPtr into routes.
		err := readConfig(*configPtr, &routes)
		if err != nil {
			log.Fatal(err)
		}
	} else if *pathPtr != "" {
		routes = append(routes, route{
			Path:     *pathPtr,
			Endpoint: *endpointPtr,
		})
	} else {
		fmt.Println("Expected either -path or -config (run static -help).")
		os.Exit(1)
	}

	switch *servicePtr {
	case "dropbox":
		initDropbox()
	case "drive":
		// TODO
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("Listening on port %d.", *portPtr)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *portPtr), nil))
}

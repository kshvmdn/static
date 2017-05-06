package main

import (
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stacktic/dropbox"
)

func initDropbox() error {
	db = dropbox.NewDropbox()
	err := db.SetAppInfo(os.Getenv("DROPBOX_CLIENT_KEY"), os.Getenv("DROPBOX_CLIENT_SECRET"))
	if err != nil {
		return err
	}
	db.SetAccessToken(os.Getenv("DROPBOX_ACCESS_TOKEN"))

	for _, route := range routes {
		endpoint := route.Endpoint
		if endpoint != "/" {
			endpoint = "/" + route.Endpoint
		}
		dr := dropboxRoute{Route: &route, Payload: &payload{}}
		go startLongPoll(&dr)
		http.HandleFunc(endpoint, makeHandler(&dr))
	}

	return nil
}

type dropboxRoute struct {
	Route   *route
	Payload *payload
	Entry   *dropbox.Entry
}

func (d *dropboxRoute) route() *route      { return d.Route }
func (d *dropboxRoute) setRoute(rt *route) { d.Route = rt }

func (d *dropboxRoute) payload() *payload { return d.Payload }

func (d *dropboxRoute) fetchFile(key string) error {
	entry, err := db.Metadata(key, false, false, "", "", 1)
	if err != nil {
		httperr, ok := err.(*dropbox.Error)
		if ok && httperr.StatusCode == http.StatusNotFound {
			d.Payload.notFound = true
			return nil
		}
		return err
	}
	// TODO: handle directories?
	if entry.IsDir {
		return errors.New("expected file, got directory")
	}

	d.Payload.lastFetch = time.Now()
	if d.Entry != nil && d.Entry.Revision == entry.Revision {
		c.Set(key, d.Route, cache.DefaultExpiration)
		return nil
	}
	d.Entry = entry

	rd, _, err := db.Download(key, "", 0)
	if err != nil {
		return err
	}
	defer rd.Close()

	d.Payload.data, err = ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	c.Set(key, d.Route, cache.DefaultExpiration)
	return nil
}

func (d *dropboxRoute) write(w http.ResponseWriter, r *http.Request) {
	if d.Payload.notFound {
		http.Error(w, "path not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(int(d.Entry.Bytes)))
	w.Header().Set("Content-Type", d.Entry.MimeType)
	w.Header().Set("ETag", d.Entry.Revision)
	w.Header().Set("Last-Modified", time.Time(d.Entry.Modified).Format(http.TimeFormat))
	w.Write(d.Payload.data)
}

func (d *dropboxRoute) longPoll() error {
	path := d.Route.Path
	if path[0] != '/' {
		path = "/" + path
	}

	lc, err := db.LatestCursor(path, false)
	if err != nil {
		return err
	}

	dp, err := db.LongPollDelta(lc.Cursor, 30)
	if err != nil {
		return err
	}

	d.Payload.lastModified = time.Now()
	max := math.Max(float64(*sleepPtr), float64(dp.Backoff))
	time.Sleep(time.Second * time.Duration(max))
	return nil
}

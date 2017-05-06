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

type dropboxRoute struct {
	r     *route
	p     *payload
	files map[string]*file
}

func initDropbox() error {
	db = dropbox.NewDropbox()
	err := db.SetAppInfo(os.Getenv("DROPBOX_CLIENT_KEY"),
		os.Getenv("DROPBOX_CLIENT_SECRET"))
	if err != nil {
		return err
	}
	db.SetAccessToken(os.Getenv("DROPBOX_ACCESS_TOKEN"))

	for _, route := range routes {
		endpoint := route.Endpoint
		if endpoint != "/" {
			endpoint = "/" + route.Endpoint
		}
		dr := dropboxRoute{
			r:     &route,
			p:     &payload{},
			files: make(map[string]*file, 0),
		}
		go startPoll(&dr)
		http.HandleFunc(endpoint, makeHandler(&dr))
	}

	return nil
}

func (d *dropboxRoute) route() *route            { return d.r }
func (d *dropboxRoute) setRoute(newRoute *route) { d.r = newRoute }

func (d *dropboxRoute) payload() *payload              { return d.p }
func (d *dropboxRoute) setPayload(newPayload *payload) { d.p = newPayload }

func (d *dropboxRoute) getFile(name string) *file    { return d.files[name] }
func (d *dropboxRoute) setFile(name string, f *file) { d.files[name] = f }

func (d *dropboxRoute) fetchFile(name string) error {
	entry, err := db.Metadata(name, false, false, "", "", 1)
	if err != nil {
		httperr, ok := err.(*dropbox.Error)
		if ok && httperr.StatusCode == http.StatusNotFound {
			d.files[name].notFound = true
			return nil
		}
		return err
	}
	if entry.IsDir {
		return errors.New("expected file; got directory")
	}

	d.files[name].lastFetched = time.Now()
	if d.files[name].entry != nil && d.files[name].entry.Revision == entry.Revision {
		c.Set(name, d.r, cache.DefaultExpiration)
		return nil
	}
	d.files[name].entry = entry

	rd, _, err := db.Download(name, "", 0)
	if err != nil {
		return err
	}
	defer rd.Close()

	d.files[name].data, err = ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	c.Set(name, d.r, cache.DefaultExpiration)
	return nil
}

func (d *dropboxRoute) write(w http.ResponseWriter, r *http.Request, name string) {
	if d.files[name].notFound {
		http.Error(w, "path not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(int(d.files[name].entry.Bytes)))
	w.Header().Set("Content-Type", d.files[name].entry.MimeType)
	w.Header().Set("Last-Modified", time.Time(d.files[name].entry.Modified).Format(http.TimeFormat))
	w.Write(d.files[name].data)
}

func (d *dropboxRoute) poll(key string) error {
	path := key
	if path == "" {
		return errors.New("key is empty")
	}
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

	d.p.lastModified = time.Now()

	max := math.Max(float64(*sleepPtr), float64(dp.Backoff))
	time.Sleep(time.Second * time.Duration(max))

	return nil
}

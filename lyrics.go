package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
)

type song struct {
	Artist  string
	Title   string
	Image   string
	Lyrics  string
	Credits map[string]string
	About   [2]string
}

func (s *song) parseLyrics(doc *goquery.Document) {
	doc.Find("[data-lyrics-container='true']").Each(func(i int, ss *goquery.Selection) {
		if h, err := ss.Html(); err == nil {
			s.Lyrics += h
		}
	})
}

func (s *song) parseMetadata(doc *goquery.Document) {
	artist := doc.Find("a[class*='Artist']").First().Text()
	title := doc.Find("h1[class*='Title']").First().Text()
	image, exists := doc.Find("meta[property='og:image']").Attr("content")
	if exists {
		if u, err := url.Parse(image); err == nil {
			s.Image = fmt.Sprintf("/images%s", u.Path)
		}
	}

	s.Title = title
	s.Artist = artist
}

func (s *song) parseCredits(doc *goquery.Document) {
	credits := make(map[string]string)

	doc.Find("[class*='SongInfo__Credit']").Each(func(i int, ss *goquery.Selection) {
		key := ss.Children().First().Text()
		value := ss.Children().Last().Text()
		credits[key] = value
	})

	s.Credits = credits
}

func (s *song) parseAbout(doc *goquery.Document) {
	s.About[0] = doc.Find("[class*='SongDescription__Content']").Text()
	summary := strings.Split(s.About[0], "")

	if len(summary) > 250 {
		s.About[1] = strings.Join(summary[0:250], "") + "..."
	}
}

func (s *song) parse(doc *goquery.Document) {
	s.parseLyrics(doc)
	s.parseMetadata(doc)
	s.parseCredits(doc)
	s.parseAbout(doc)
}

func lyricsHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if data, err := getCache(id); err == nil {
		render("lyrics", w, data)
		return
	}

	url := fmt.Sprintf("https://genius.com/%s-lyrics", id)
	resp, err := http.Get(url)
	if err != nil {
		write(w, http.StatusInternalServerError, []byte("can't reach genius servers"))
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		write(w, http.StatusNotFound, []byte("Not found"))
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		write(w, http.StatusInternalServerError, []byte("something went wrong"))
		return
	}

	var s song
	s.parse(doc)

	w.Header().Set("content-type", "text/html")

	render("lyrics", w, s)
	setCache(id, s)
}

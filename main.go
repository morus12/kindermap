package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	geojson "github.com/paulmach/go.geojson"
)

var (
	reg *regexp.Regexp = regexp.MustCompile(`^[^+\d]+[0-9]+`)
)

func main() {
	errlog := log.New(os.Stderr, "", 1)

	resp, err := http.Get("https://rekrutacja-zlobki.um.wroc.pl/wroclaw/zlobek/oferta")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		errlog.Panicln(err)
	}

	fc := geojson.NewFeatureCollection()

	doc.Find("#unitstable > tbody > tr").Each(func(i int, s *goquery.Selection) {

		name := s.Find("a:nth-child(1)").Text()
		addr := s.Find("a:nth-child(2)").Text()
		link, exists := s.Find("a:nth-child(1)").Attr("href")
		if addr == "" {
			return
		}

		district := s.Find(".district").Text()

		point, err := addrToPoint(addr)
		if err != nil {
			errlog.Println(err)
			return
		}

		point.SetProperty("name", name)
		point.SetProperty("district", district)
		if exists {
			point.SetProperty("description", "[[https://rekrutacja-zlobki.um.wroc.pl"+link+"|Szczegóły]]\n"+addr)
		}
		fc.AddFeature(point)
	})

	rawJSON, err := fc.MarshalJSON()
	if err != nil {
		errlog.Panicln(err)
	}

	err = ioutil.WriteFile("geo.json", rawJSON, 0664)
	if err != nil {
		errlog.Panicln(err)
	}
}

type Place struct {
	Lon string `json:"lon"`
	Lat string `json:"lat"`
}

func addrToPoint(addr string) (*geojson.Feature, error) {

	place, err := getPlace(addr)
	if err != nil {

		addr := reg.FindString(addr)
		parts := strings.Split(addr, ".")
		addr = parts[len(parts)-1]
		place, err = getPlace(addr + ", Wrocław")

		if err != nil {
			return nil, err
		}
	}

	lat, err := strconv.ParseFloat(place.Lat, 64)
	if err != nil {
		return nil, err
	}

	lon, err := strconv.ParseFloat(place.Lon, 64)
	if err != nil {
		return nil, err
	}

	return geojson.NewPointFeature([]float64{lon, lat}), nil
}

func getPlace(query string) (*Place, error) {
	qs := url.Values{
		"format":         []string{"json"},
		"q":              []string{query},
		"addressdetails": []string{"0"},
		"countrycodes":   []string{"PL"},
	}

	resp, err := http.Get("https://nominatim.openstreetmap.org/search?" + qs.Encode())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	raw, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var places []*Place

	err = json.Unmarshal(raw, &places)
	if err != nil {
		return nil, err
	}

	if len(places) == 0 {
		return nil, fmt.Errorf("not found: %v", query)
	}

	return places[0], nil
}

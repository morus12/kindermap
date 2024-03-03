package main

import (
	"encoding/csv"
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

	geojson "github.com/paulmach/go.geojson"
)

var reg *regexp.Regexp = regexp.MustCompile(`^[^+\d]+[0-9]+`)

type Place struct {
	Lon string `json:"lon"`
	Lat string `json:"lat"`
}

func main() {
	errlog := log.New(os.Stderr, "", 1)

	// Load CSV file
	file, err := os.Open("convertcsv1.csv")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	fc := geojson.NewFeatureCollection()

	for _, record := range records {
		// Assuming the structure of CSV records
		if len(record) < 11 {
			continue
		}
		name := record[1]
		addr := record[3]

		district := record[10]

		point, err := addrToPoint(addr)
		if err != nil {
			errlog.Println(err)
			continue
		}

		point.SetProperty("name", name)
		point.SetProperty("district", district)

		fc.AddFeature(point)
	}

	rawJSON, err := fc.MarshalJSON()
	if err != nil {
		errlog.Panicln(err)
	}

	err = ioutil.WriteFile("geo.json", rawJSON, 0o664)
	if err != nil {
		errlog.Panicln(err)
	}
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

	fmt.Println(qs.Encode(), places[0])
	return places[0], nil
}

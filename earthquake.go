package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type Quakes struct {
	Type     string `json:"type"`
	Metadata struct {
		Generated int64  `json:"generated"`
		URL       string `json:"url"`
		Title     string `json:"title"`
		Status    int    `json:"status"`
		API       string `json:"api"`
		Count     int    `json:"count"`
	} `json:"metadata"`
	Features []struct {
		Type       string `json:"type"`
		Properties struct {
			Mag     float64     `json:"mag"`
			Place   string      `json:"place"`
			Time    int64       `json:"time"`
			Updated int64       `json:"updated"`
			Tz      int         `json:"tz"`
			URL     string      `json:"url"`
			Detail  string      `json:"detail"`
			Felt    interface{} `json:"felt"`
			Cdi     interface{} `json:"cdi"`
			Mmi     interface{} `json:"mmi"`
			Alert   interface{} `json:"alert"`
			Status  string      `json:"status"`
			Tsunami int         `json:"tsunami"`
			Sig     int         `json:"sig"`
			Net     string      `json:"net"`
			Code    string      `json:"code"`
			Ids     string      `json:"ids"`
			Sources string      `json:"sources"`
			Types   string      `json:"types"`
			Nst     int         `json:"nst"`
			Dmin    float64     `json:"dmin"`
			Rms     float64     `json:"rms"`
			Gap     int         `json:"gap"`
			MagType string      `json:"magType"`
			Type    string      `json:"type"`
			Title   string      `json:"title"`
		} `json:"properties"`
		Geometry struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		ID string `json:"id"`
	}
	Bbox    []float64 `json:"bbox"`
	Special struct {
		Worstregion    string
		Countrymost    string
		Countrymostint int
		Meanmag        float64
		Ort            []string
		Country        string
	}
}

func main() {
	var quakes Quakes

	//parse template
	tpl, err := template.ParseFiles("tpl.gohtml", "quake.gohtml", "welcome.gohtml")
	if err != nil {
		log.Fatalln(err)
	}

	//get data
	data := getRecordsstdin("https://earthquake.usgs.gov/fdsnws/event/1/query?format=geojson&")

	//decode data
	_ = json.Unmarshal(data, &quakes)

	//Filling in Special struct
	quakes.Special.Meanmag = meanmag(quakes)
	mapcountry := countrycount(quakes)
	quakes.Special.Countrymost, quakes.Special.Countrymostint = countrymost(mapcountry)
	for _, value := range quakes.Features {
		ort := getcountry(value.Properties.Place)
		if contains(quakes.Special.Ort, ort) == false {
			quakes.Special.Ort = append(quakes.Special.Ort, ort)
		}
	}

	router := httprouter.New()
	router.GET("/quakes/", func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

		err := tpl.ExecuteTemplate(w, "tpl.gohtml", quakes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatalln(err)
		}
	})
	router.GET("/quakes/:ort", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		ort := p.ByName("ort")
		var quakes2 Quakes
		quakes2.Type = quakes.Type
		quakes2.Metadata = quakes.Metadata
		quakes2.Bbox = quakes.Bbox
		quakes2.Special = quakes.Special
		quakes2.Special.Country = ort

		for _, quake := range quakes.Features {
			if ort == getcountry(quake.Properties.Place) {
				quakes2.Features = append(quakes2.Features, quake)
			}
		}
		err := tpl.ExecuteTemplate(w, "quake.gohtml", quakes2)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatalln(err)
		}
	})

	http.ListenAndServe(":8080", router)

}

func contains(slice []string, word string) bool {
	for _, value := range slice {
		if word == value {
			return true
		}
	}
	return false
}

func getcountry(place string) string {
	var country string
	if strings.Contains(place, ", ") {
		whole := strings.Split(place, ", ")
		country = whole[len(whole)-1]
	} else {
		country = place
	}
	return country
}

//data with date from stdin
func getRecordsstdin(url string) []byte {
	fmt.Println("Gib Anfangs- und Enddatum an. (dd.mm.yyyy)")
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Anfangsdatum: ")
	scanner.Scan()
	start := parsedate(scanner.Text())

	fmt.Print("Enddatum: ")
	scanner.Scan()
	ende := parsedate(scanner.Text())

	resp, err := http.Get(url + "starttime=" + start + "&endtime=" + ende)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	return data
}

func getRecords(url string, start string, ende string) []byte {
	start = parsedate(start)
	ende = parsedate(ende)

	resp, err := http.Get(url + "starttime=" + start + "&endtime=" + ende)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	return data
}

//date from dd.mm.yyyy to yyyy-mm-dd
func parsedate(datum string) string {
	date := strings.Split(datum, ".")
	return date[2] + "-" + date[1] + "-" + date[0]

}

//mean magnitude
func meanmag(quakes Quakes) float64 {
	var sum float64
	for _, quake := range quakes.Features {
		sum += quake.Properties.Mag
	}
	return sum / float64(quakes.Metadata.Count)
}

//count number of earthquakes for each country
func countrycount(quakes Quakes) map[string]int {
	counts := make(map[string]int)
	for _, quake := range quakes.Features {
		if strings.Contains(quake.Properties.Place, ",") {
			country := strings.Split(quake.Properties.Place, ",")
			counts[country[len(country)-1]]++
		} else {
			country := quake.Properties.Place
			counts[country]++
		}
	}
	return counts
}

//country with most earthquakes and number of that country
func countrymost(counts map[string]int) (string, int) {
	country := ""
	num := 0
	for k, v := range counts {
		if v > num {
			country = k
			num = v
		}
	}
	return country, num
}

//all coordinates of quakes in a single slice [lat,long]
func getcoordinates(quakes Quakes) [][]float64 {
	var geomplaces [][]float64
	for _, quake := range quakes.Features {
		point := []float64{quake.Geometry.Coordinates[1], quake.Geometry.Coordinates[0]}
		geomplaces = append(geomplaces, point)
	}
	return geomplaces
}

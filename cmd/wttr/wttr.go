package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultCity = "Berkeley"
	defaultURL  = "https://wttr.in"
)

var defaultClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Options struct {
	City   string
	Params url.Values
}

type WeatherClient struct {
	httpClient *http.Client
	options    *Options
}

func NewWeatherClient(options *Options) *WeatherClient {
	if options.City == "" {
		options.City = defaultCity
	}
	if options.Params == nil {
		options.Params = url.Values{
			"q": {""},
			"0": {""},
		}
	}
	return &WeatherClient{
		httpClient: defaultClient,
		options:    options,
	}
}

func (wc *WeatherClient) GetWeather() error {
	queryURL := fmt.Sprintf("%s/%s", defaultURL, wc.options.City)
	request, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return err
	}
	request.URL.RawQuery = wc.options.Params.Encode()

	response, err := wc.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error: %s returned status %s", defaultURL, response.Status)
	}

	return displayWeather(response.Body)
}

func displayWeather(reader io.Reader) error {
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	fmt.Print(string(body))
	return nil
}

func main() {
	location := flag.String("location", defaultCity, "Location for the weather report")
	onlyCurrent := flag.Bool("0", true, "Only current weather")
	superQuietVersion := flag.Bool("Q", true, "Superquiet version")
	forceANSI := flag.Bool("na", false, "Request non-ANSI output format. (HTML)")
	currentAndToday := flag.Bool("1", false, "Current weather + today's forecast")
	currentTodayTomorrow := flag.Bool("2", false, "Current weather + today's + tomorrow's forecast")
	restrictGlyphs := flag.Bool("d", false, "Restrict output to standard console font glyphs")
	narrowVersion := flag.Bool("n", false, "Narrow version")
	quietVersion := flag.Bool("q", false, "Quiet version")
	noTermSeqs := flag.Bool("T", false, "Switch terminal sequences off")
	disableDefaults := flag.Bool("nd", false, "Disable default options")

	flag.Parse()

	// Determine if any flags other than the location were provided, if so, no default flags are used
	otherFlagsSet := flag.NFlag() > 1 || *disableDefaults

	params := url.Values{}
	params.Add("F", "")
	if !*forceANSI {
		params.Add("A", "")
	}
	if *onlyCurrent && !otherFlagsSet {
		params.Add("0", "")
	}
	if *currentAndToday {
		params.Add("1", "")
	}
	if *currentTodayTomorrow {
		params.Add("2", "")
	}
	if *restrictGlyphs {
		params.Add("d", "")
	}
	if *narrowVersion {
		params.Add("n", "")
	}
	if *quietVersion {
		params.Add("q", "")
	}
	if *superQuietVersion && !otherFlagsSet {
		params.Add("Q", "")
	}
	if *noTermSeqs {
		params.Add("T", "")
	}

	clientOptions := &Options{
		City:   *location,
		Params: params,
	}
	client := NewWeatherClient(clientOptions)
	if err := client.GetWeather(); err != nil {
		log.Fatalln(err)
	}
}
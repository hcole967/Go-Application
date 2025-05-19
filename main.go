// Made by Harrison Cole 10632167
// the purpose of this code is to open and take the data from a website to show the weather for the upcoming week from a city of choice
// originally, the goal was to also include humidity in the data, however the only available data is hourly which will not show for the 7 day forecast
// therefore i chose to omit humidity and replace it with wind/weatherstatus(sunny etc) instead

// Upon first opening the file, my directory would default to the parent folder directory 'Go Assignment'
// and i would have to change the directory using 'cd Weather_Application_CSP3341' before using 'go run main.go' to run the program

// I found the .json source code for the website through the following method:
// find website, right click inside website, press inspect, navigate to 'network', reload page, search for 'json', click the link that shows up

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Location struct {
	Lat string
	Lon string
	Err error
}

// found the .json files linked to open source street map website so I can put the data in a struct
func lonlat(cityInput string, ch chan<- Location) { //can be done the same as a sequential function, but I wanted to show goroutines and channels in use
	encodedCity := url.QueryEscape(cityInput)
	cityURL := "https://nominatim.openstreetmap.org/search?q=" + encodedCity + "&format=json&addressdetails=1"

	resp, err := http.Get(cityURL)
	if err != nil {
		ch <- Location{"", "", fmt.Errorf("request error: %v", err)}
		return
	}
	defer resp.Body.Close()

	var geo []struct {
		Lon     string `json:"lon"`
		Lat     string `json:"lat"`
		Address struct {
			City    string `json:"city"`
			Town    string `json:"town"`
			Village string `json:"village"`
			Country string `json:"country"`
		} `json:"address"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		ch <- Location{"", "", fmt.Errorf("decode error: %v", err)}
		return
	}

	if len(geo) == 0 {
		ch <- Location{"", "", fmt.Errorf("no results found")}
		return
	}

	// find the city first, if not check if it's a town, then a village
	addr := geo[0].Address
	city := addr.City
	if city == "" {
		if addr.Town != "" {
			city = addr.Town
		} else {
			city = addr.Village
		}
	}

	fmt.Println("City:", city, ",", addr.Country)
	ch <- Location{geo[0].Lat, geo[0].Lon, nil}
}

// found the .json files (API) linked to open source weather website so I can put the data in a struct
func getWeather(lat, lon string, done chan<- bool) { //can be done without channels but wanted to include them for concurrency showcasing
	weatherURL := "https://api.open-meteo.com/v1/forecast?latitude=" + lat + "&longitude=" + lon + "&&daily=temperature_2m_max,temperature_2m_min,weathercode,precipitation_probability_max&current_weather=true&timezone=auto&timeformat=iso8601"

	resp, err := http.Get(weatherURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Weather request error: %v\n", err)
		done <- true
		return
	}
	defer resp.Body.Close()

	// Wanted to get different weather codes (rainy, sunny, etc) for a more accurate weather app. Unfortunately there's 28 codes
	// The website https://open-meteo.com/en/docs shows all of the weather codes and their meanings so I have taken these 1:1 and made a struct
	var weatherCodeMap = map[int]string{
		0: "Clear sky", 1: "Mainly clear", 2: "Partly cloudy", 3: "Overcast", 45: "Fog", 48: "Depositing rime fog", 51: "Light drizzle", 53: "Moderate drizzle",
		55: "Dense drizzle", 56: "Light freezing drizzle", 57: "Dense freezing drizzle", 61: "Slight rain", 63: "Moderate rain", 65: "Heavy rain", 66: "Light freezing rain",
		67: "Heavy freezing rain", 71: "Slight snow fall", 73: "Moderate snow fall", 75: "Heavy snow fall", 77: "Snow grains", 80: "Slight rain showers",
		81: "Moderate rain showers", 82: "Violent rain showers", 85: "Slight snow showers", 86: "Heavy snow showers", 95: "Thunderstorm", 96: "Thunderstorm with slight hail",
		99: "Thunderstorm with heavy hail",
	}

	var weather struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			Windspeed   float64 `json:"windspeed"`
			WindDir     float64 `json:"winddirection"`
			Time        string  `json:"time"`
			Is_day      int     `json:"is_day"`
		} `json:"current_weather"`
		Daily struct {
			Time                     []string  `json:"time"`
			TemperatureMax           []float64 `json:"temperature_2m_max"`
			TemperatureMin           []float64 `json:"temperature_2m_min"`
			WeatherCode              []int     `json:"weathercode"`
			PrecipitationProbability []int     `json:"precipitation_probability_max"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		fmt.Fprintf(os.Stderr, "JSON decode error: %v\n", err)
		done <- true
		return
	}

	fmt.Println("\n----- Current Weather -----")
	fmt.Println("\nCurrent temperature:", weather.CurrentWeather.Temperature, "°C")
	fmt.Println("Windspeed:", weather.CurrentWeather.Windspeed, "km/h", weather.CurrentWeather.WindDir, "°")
	fmt.Println("Time:", weather.CurrentWeather.Time)
	if weather.CurrentWeather.Is_day == 1 {
		fmt.Println("It is currently day time")
	} else {
		fmt.Println("It is currently night time")
	}

	fmt.Println("\n----- 7 Day Forecast -----")
	for i := 0; i < len(weather.Daily.Time); i++ {
		// change date from '2025-05-15' to format: 'dayOfWeek, DD/MM'
		date := weather.Daily.Time[i]
		parsedDate, err := time.Parse("2006-01-02", date) // the date reference that Go uses is from 2006
		if err != nil {
			fmt.Println("Date parse error:", err)
			continue
		}
		betterDate := parsedDate.Format("Monday 02/01")

		// use the weather code struct to match to forecast
		desc := weatherCodeMap[weather.Daily.WeatherCode[i]]

		fmt.Printf("\nDay %d - %s\n", i+1, betterDate)
		fmt.Printf("  - High: %.1f°C\n", weather.Daily.TemperatureMax[i])
		fmt.Printf("  - Low: %.1f°C\n", weather.Daily.TemperatureMin[i])
		fmt.Printf("  - Weather: %s\n", desc)
		fmt.Printf("  - Rain chance: %d%%\n", weather.Daily.PrecipitationProbability[i])
	}
	done <- true
}

// wanted to keep running main as long as user inputs 'y', so made this to function as the main runner
func runWeatherConcurrent() {
	var city string
	fmt.Print("Enter a city: ")
	fmt.Scanln(&city)

	locationChan := make(chan Location)
	weatherDone := make(chan bool)

	go lonlat(city, locationChan) // start location goroutine

	loc := <-locationChan // waiting
	if loc.Err != nil {
		fmt.Fprintf(os.Stderr, "Error getting location: %v\n", loc.Err)
		return
	}

	go getWeather(loc.Lat, loc.Lon, weatherDone) // start weather goroutine

	<-weatherDone // waiting
}

func main() {
	for {
		runWeatherConcurrent()

		var cont string
		fmt.Print("\nDo you want to continue? (y/n): ")
		fmt.Scanln(&cont)

		if cont == "y" {
			continue
		} else if cont == "n" {
			fmt.Println("Exiting...")
			break
		} else {
			fmt.Println("Invalid input, exiting system")
			break
		}
	}
}

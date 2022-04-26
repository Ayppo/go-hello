package main

import "net/http"
import "strings"
import "encoding/json"
import "log"

type weatherData struct {
    Name string `json:"name"`
    Main struct {
        Kelvin float64 `json:"temp"`
    } `json:"main"`
}

func query(city string) (weatherData, error) {
    resp, err := http.Get("https://api.openweathermap.org/data/2.5/weather?appid=110e7eb4f6b336bb2e97c0491731cada&q=" + city)
    if err != nil {
        return weatherData {}, err
    }
    defer resp.Body.Close()
    var d weatherData
    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return weatherData {}, err
    }
    return d, nil
}

type weatherProvider interface {
    temperature (city string) (float64, error)
}
type openWeatherMap struct {}

func (w openWeatherMap) temperature(city string) (float64, error) {
    resp, err := http.Get("https://api.openweathermap.org/data/2.5/weather?appid=110e7eb4f6b336bb2e97c0491731cada&q=" + city)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }
    log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
    return d.Main.Kelvin, nil
}

type weatherUnderground struct {
    apiKey string
}

func (w weatherUnderground) temperature(city string) (float64, error) {
    resp, err := http.Get("www.baidu.com" + city + w.apiKey)
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Observation struct {
            Celsius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, nil
    }
    kelvin := d.Observation.Celsius + 273.15
    log.Printf("weartherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}

type multiWeatherProvider []weatherProvider
//func temperature(city string, providers ...weatherProvider) (float64, error) {
func (w multiWeatherProvider) temperature(city string) (float64, error) {
    temps := make(chan float64, len(w))
    errs := make(chan error, len(w))

    for _, provider := range w {
        go func (p weatherProvider) {
            k, err := p.temperature(city)
            if err != nil {
                errs <- err
                return
            }
            temps <- k
        }(provider)
    }

    sum := 0.0
    for i := 0; i < len(w); i++ {
        select {
        case temp := <- temps:
            sum += temp
        case err := <- errs:
            return 0, err
        }
    }
    return sum / float64(len(w)), nil
}


func main() {
    http.HandleFunc("/", hello)
    http.HandleFunc("/weather/", weather)
    http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello"))
}

func weather(w http.ResponseWriter, r *http.Request) {
    city := strings.SplitN(r.URL.Path, "/", 3)[2]
    data, err := query(city)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    json.NewEncoder(w).Encode(data)
}

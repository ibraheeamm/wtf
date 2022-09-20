package nextbus

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/logger"
	"github.com/wtfutil/wtf/view"
)

// Widget is the container for your module's data
type Widget struct {
	view.TextWidget

	settings *Settings
}

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, pages, settings.common),

		settings: settings,
	}
	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) content() string {
	return getNextBus("", "")
}

type AutoGenerated struct {
	Copyright   string      `json:"copyright"`
	Predictions Predictions `json:"predictions"`
}

type Prediction struct {
	AffectedByLayover string `json:"affectedByLayover"`
	Seconds           string `json:"seconds"`
	TripTag           string `json:"tripTag"`
	Minutes           string `json:"minutes"`
	IsDeparture       string `json:"isDeparture"`
	Block             string `json:"block"`
	DirTag            string `json:"dirTag"`
	Branch            string `json:"branch"`
	EpochTime         string `json:"epochTime"`
	Vehicle           string `json:"vehicle"`
}

type Direction struct {
	PredictionRaw json.RawMessage `json:"prediction"`
	Title         string          `json:"title"`
}

type Predictions struct {
	RouteTag    string    `json:"routeTag"`
	StopTag     string    `json:"stopTag"`
	RouteTitle  string    `json:"routeTitle"`
	AgencyTitle string    `json:"agencyTitle"`
	StopTitle   string    `json:"stopTitle"`
	Direction   Direction `json:"direction"`
}

// https://webservices.umoiq.com/service/publicJSONFeed?command=predictions&a=ttc&stopId=14646
// https://webservices.umoiq.com/service/publicJSONFeed?command=predictions&a=ttc&r=320&stopId=1669
func getNextBus(route string, stopID string) string {
	// route = "ttc"
	// stopID = "14646"
	url := "https://webservices.umoiq.com/service/publicJSONFeed?command=predictions&a=ttc&stopId=14646"
	// url := "https://webservices.umoiq.com/service/publicJSONFeed?command=predictions&a=ttc&r=320&stopId=1669"
	resp, err := http.Get(url)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to make request to TTC. ERROR: %s", err))
		return "ERROR REQ"
	}
	body, readErr := io.ReadAll(resp.Body)

	if (readErr) != nil {
		return "ERROR"
	}

	resp.Body.Close()

	// body := []byte(`{"copyright":"All data copyright Toronto Transit Commission 2022.","predictions":{"routeTag":"14","stopTag":"14168","routeTitle":"14-Glencairn","agencyTitle":"Toronto Transit Commission","stopTitle":"Davisville Station","direction":{"prediction":{"affectedByLayover":"true","seconds":"3436","tripTag":"44229096","minutes":"57","isDeparture":"true","block":"14_3_32","dirTag":"14_1_14","branch":"14","epochTime":"1663629540000","vehicle":"1416"},"title":"West - 14 Glencairn towards Caledonia"}}}`)
	// body := []byte(`{"copyright":"All data copyright Toronto Transit Commission 2022.","predictions":{"routeTag":"14","stopTag":"14168","routeTitle":"14-Glencairn","agencyTitle":"Toronto Transit Commission","stopTitle":"Davisville Station","direction":{"prediction":[{"affectedByLayover":"true","seconds":"834","tripTag":"44229082","minutes":"13","isDeparture":"true","block":"14_1_10","dirTag":"14_1_14","branch":"14","epochTime":"1663642860000","vehicle":"1143"},{"affectedByLayover":"true","seconds":"2147","tripTag":"44229083","minutes":"35","isDeparture":"true","block":"14_3_32","dirTag":"14_1_14","branch":"14","epochTime":"1663644172665","vehicle":"1416"}],"title":"West - 14 Glencairn towards Caledonia"}}}`)

	var parsedResponse AutoGenerated

	// partial unmarshal, we don't have r.Predictions.Direction.PredictionRaw <- YET
	unmarshalError := json.Unmarshal(body, &parsedResponse)
	if unmarshalError != nil {
		log.Fatal(err)
	}

	parseType := ""
	// hacky, try object parse first
	item := Prediction{}
	if err := json.Unmarshal(parsedResponse.Predictions.Direction.PredictionRaw, &item); err == nil {
		parseType = "object"
	}

	// if object parse failed, it probably means we have an array
	items := []Prediction{}
	if err := json.Unmarshal(parsedResponse.Predictions.Direction.PredictionRaw, &items); err == nil {
		parseType = "array"
	}

	//
	// build the final string
	finalStr := ""
	if parseType == "array" {
		for _, itm := range items {
			seconds, _ := strconv.Atoi(itm.Seconds)
			minutes, _ := strconv.Atoi(itm.Minutes)
			seconds = seconds % 60
			finalStr += fmt.Sprintf("%s [%02d:%02d] Bus: %s\n", parsedResponse.Predictions.RouteTitle, minutes, seconds, itm.Vehicle)
		}
	} else {
		seconds, _ := strconv.Atoi(item.Seconds)
		minutes, _ := strconv.Atoi(item.Minutes)

		seconds = seconds % 60
		finalStr += fmt.Sprintf("%s [%02d:%02d] Bus: %s\n", parsedResponse.Predictions.RouteTitle, minutes, seconds, item.Vehicle)
	}

	return finalStr
}
func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), false
	})
}

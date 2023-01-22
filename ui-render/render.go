package uirender

import (
	"html/template"
	"net/http"

	"github.com/nax11/tinkoff_bot_public/strategy"
	"github.com/nax11/tinkoff_bot_public/ui-render/models"
)

func RunUI(reportParams strategy.ReportParams) {
	htmlData := prepareHtml(reportParams)
	buildHtml(htmlData)
}

func prepareHtml(reportParams strategy.ReportParams) models.HtmlData {
	result := models.HtmlData{
		MarketHigh: []models.Item{},
		MarketLow:  []models.Item{},
		CalcPrice:  []models.CalcItem{},
	}
	for i, item := range reportParams.AnalyzedData {
		marketHighItem := models.Item{
			D1: i,
			V1: item.MarketHighPrice,
		}
		result.MarketHigh = append(result.MarketHigh, marketHighItem)

		marketLowItem := models.Item{
			D1: i,
			V1: item.MarketLowPrice,
		}
		result.MarketLow = append(result.MarketLow, marketLowItem)

		if item.CalculatedBuyPrice != 0 {
			calcItem := models.CalcItem{
				D1:   i,
				Buy:  item.CalculatedBuyPrice,
				Sell: item.CalculatedSellPrice,
			}
			result.CalcPrice = append(result.CalcPrice, calcItem)
		}
	}
	return result
}

func buildHtml(htmlData models.HtmlData) {
	tmpl := template.Must(template.ParseFiles("ui/index.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, htmlData)
	})

	http.HandleFunc("/jquery.min.js", sendJqueryJs)
	http.HandleFunc("/jquery.canvasjs.min.js", sendCanvasJs)

	http.ListenAndServe(":8080", nil)
}

func sendJqueryJs(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/js/jquery.min.js")
}

func sendCanvasJs(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/js/jquery.canvasjs.min.js")
}

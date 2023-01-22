package models

type HtmlData struct {
	MarketHigh []Item
	MarketLow  []Item
	CalcPrice  []CalcItem
}

type Item struct {
	D1 int
	V1 float64
}

type CalcItem struct {
	D1   int
	Buy  float64
	Sell float64
}

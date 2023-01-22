package models

type AverageSlice []float64

func (a AverageSlice) AveragePrice() float64 {
	total := 0.0
	for _, item := range a {
		total += item
	}
	return total / float64(len(a))
}

type SimulateOrder struct {
	BuyPrice    float64
	BuySum      float64
	Qty         int32
	SellPrice   float64
	SellSum     float64
	Profit      float64
	IsPurchased bool
	IsSold      bool
}

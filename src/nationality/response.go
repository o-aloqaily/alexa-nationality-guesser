package nationality

type Response struct {
	Predictions []Prediction `json:"country"`
}

type Prediction struct {
	Country_id  string
	Probability float64
}

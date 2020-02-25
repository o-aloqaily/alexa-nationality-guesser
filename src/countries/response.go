package countries

type Country []struct {
	Demonym string `json:"demonym"`
	Code    string `json:"alpha2Code"`
}

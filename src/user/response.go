package user

type User struct {
	Attributes []Attribute `json:"UserAttributes"`
}

type Attribute struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

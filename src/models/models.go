package models

type Conf struct {
	Server       string
	Login        string
	Domain       string
	Pass         string
	Threads      int
	FieldProject string
	FieldExpire  string
	Origins      map[string]int
}

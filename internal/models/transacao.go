package models

import "github.com/google/uuid"

type Lancamento struct {
	UUID		uuid.UUID	`json:"id"`
	Descricao	string		`json:"descricao"`
	Valor		float64		`json:"valor"`
	Data		string		`json:"data"`
	Categoria	string		`json:"categoria"`
	Ativo bool `json:"ativo"`
}

package main

import (
	"context"
	"fmt"
	"os"
	"prisma/internal/database"
	"prisma/internal/models"

	"github.com/google/uuid"
)

// App struct
type App struct {
	ctx context.Context
	db	*database.Repository
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Inicializa o repositório do banco de dados
	repo, err := database.NewRepository()
	if err != nil {
		// TODO: Em vez de um Println, usar o Runtime
		// do Wails para mostrar um diálogo de erro fatal.
		fmt.Printf("ERRO FATAL AO INICIAR O BANCO: %v\n", err)
		os.Exit(1)
	}

	// Injeta o repositório na struct App
	a.db = repo
}

// SalvarLancamento é a função "ponte" que o Vue irá chamar.
// TODO: Validar os dados que chegam antes de repassar para a função de cadastro
func (a *App) SalvarLancamento(descricao string, valor float64, data string, categoria string) (string, error) {
	novoUUID := uuid.New()

	// 2. Criar o modelo de dados
	novoLancamento := models.Lancamento{
		UUID:        novoUUID,
		Descricao: descricao,
		Valor:     valor,
		Data:      data,
		Categoria: categoria,
		Ativo: true,
	}

	// 3. Chamar a camada de "backend" (o Repositório)
	err := a.db.SalvarLancamento(novoLancamento)
	if err != nil {
		// Retorna o erro para o Vue.js
		return "", fmt.Errorf("erro ao salvar no banco: %w", err)
	}

	// Retorna uma string de sucesso e 'nil' para o erro
	return "Lançamento salvo com sucesso!", nil
}

// Função "ponte" que o Vue irá chamar para realizar o Soft Delete
// Recebe o UUID (como string) e repassa para o repositório
func (a *App) SoftDeleteLancamento(uuid string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("UUID fornecido está vazio")
	}

	// Chamar a camada de "backend" (o Repositório)
	err := a.db.SoftDeleteLancamento(uuid)
	if err != nil {
		return "", err
	}

	// Retorna uma string de sucesso e 'nil' para o erro
	return "Lançamento arquivado com sucesso!", nil
}

// Busca um Lançamento ativo pelo seu UUID
func (a *App) BuscarLancamentoPorUUID(uuid string) (models.Lancamento, error) {
	return a.db.BuscarLancamentoPorUUID(uuid)
}

// BuscarLancamentos é a "ponte" para a busca dinâmica.
// Ex: { "descricao": "mercado", "categoria": "Despesas Variáveis" }
func (a *App) BuscarLancamentos(filtros models.LancamentoFiltros) ([]models.Lancamento, error) {
	return a.db.BuscarLancamentos(filtros)
}

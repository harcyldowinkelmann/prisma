package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"prisma/internal/models"

	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

// Função privata que encontra o local seguro para salvar o 'prisma.db'
func getDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível encontrar o diretório de config: %w", err)
	}

	// O caminho será .../AppData/Roaming/Prisma/
	appDataDir := filepath.Join(configDir, "Prisma")

	// Garante que o diretório exista
	if err = os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar o diretório de dados: %w", err)
	}

	// Retorna o caminho completo para o arquivo
	return filepath.Join(appDataDir, "prisma.db"), nil
}

func NewRepository() (*Repository, error) {
	dbPath, err := getDatabasePath()
	if err != nil {
		return nil, err
	}

	// sql.Open vai criar o 'prisma.db' se não existir
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o banco: %w", err)
	}

	// Cria um repositório temporário para chamar a migração
	repo := &Repository{db: db}
	if err = repo.initTables(); err != nil {
		return nil, fmt.Errorf("erro ao inicializar tabelas: %w", err)
	}

	fmt.Println("Banco de dados iniciado com sucesso em: ", dbPath)
	return repo, nil
}

// initTables executa a migração (schema) do banco
func (r *Repository) initTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS lancamentos (
			uuid TEXT PRIMARY KEY,
			descricao TEXT NOT NULL,
			valor REAL NOT NULL,
			data TEXT NOT NULL,
			categoria TEXT NOT NULL,
			ativo INTEGER NOT NULL DEFAULT 1
		);
	`

	_, err := r.db.Exec(query)
	return err
}

// Recebe os dados do lançamento e salva no banco
func (r *Repository) SalvarLancamento(l models.Lancamento) error {
	query := `
		INSERT INTO lancamentos (uuid, descricao, valor, data, categoria, ativo)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, l.UUID, l.Descricao, l.Valor, l.Data, l.Categoria, l.Ativo)
	if err != nil {
		return fmt.Errorf("erro ao inserir lançamento: %w", err)
	}

	return nil
}

// Executa um "soft delete" atualizando o campo 'ativo' para 0 (false)
func (r *Repository) SoftDeleteLancamento(uuid string) error {
	query := `
		UPDATE lancamentos
		SET ativo = 0
		WHERE uuid = ?;
	`

	// Executa o SQL, passando o UUID como argumento.
	res, err := r.db.Exec(query, uuid)
	if err != nil {
		return fmt.Errorf("erro ao executar o soft delete: %w", err)
	}

	// Verificando se algum registro foi afetado, confirmando o SoftDelete
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao verificar linhas afetadas: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("nenhum lançamento encontrado com o UUID fornecido: %s", uuid)
	}

	return nil
}

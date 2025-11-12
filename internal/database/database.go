package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"prisma/internal/models"
	"strings"

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

// Busca lançamentos ativos com base em seus filtros opcionais
func (r *Repository) BuscarLancamentos(filtros models.LancamentoFiltros) ([]models.Lancamento, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT * FROM lancamentos WHERE ativo = 1")

	// args (argumentos) são os valores para os '?'
	var args []interface{}
	
	// Filtro por Descricao (usando LIKE)
	if filtros.Descricao != nil && *filtros.Descricao != "" {
		queryBuilder.WriteString(" AND descricao LIKE ?")
		args = append(args, "%" + *filtros.Descricao + "%")
	}

	// Filtro por Valor (exato)
	if filtros.Valor != nil {
		queryBuilder.WriteString(" AND valor = ?")
		args = append(args, *filtros.Valor)
	}

	// Filtro por Data (exata)
	if filtros.Data != nil && *filtros.Data != "" {
		queryBuilder.WriteString(" AND data = ?")
		args = append(args, *filtros.Data)
	}

	// Filtro por Categoria (exata)
	if filtros.Categoria != nil && *filtros.Categoria != "" {
		queryBuilder.WriteString(" AND categoria = ?")
		args = append(args, *filtros.Categoria)
	}

	queryBuilder.WriteString(" ORDER BY data DESC;")

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar busca dinâmica: %w", err)
	}
	defer rows.Close()

	return r.scanLancamentos(rows)
}

// Busca um Lançamento ativo pelo seu UUID
func (r *Repository) BuscarLancamentoPorUUID(uuid string) (models.Lancamento, error) {
	query := "SELECT * FROM lancamentos WHERE uuid = ? AND ativo = 1;"

	var l models.Lancamento

	err := r.db.QueryRow(query, uuid).Scan(&l.UUID, &l.Descricao, &l.Valor, &l.Data, &l.Categoria, &l.Ativo)
	if err != nil {
		if err == sql.ErrNoRows {
			return l, fmt.Errorf("nenhum lançamento ativo encontrado com o UUID: %s", uuid)
		}
		return l, fmt.Errorf("erro ao buscar por UUID: %w", err)
	}

	return l, nil
}

// --- FUNÇÕES AUXILIARES ---

// scanLancamentos (Helper)
func (r *Repository) scanLancamentos(rows *sql.Rows) ([]models.Lancamento, error) {
	var lancamentos []models.Lancamento

	for rows.Next() {
		var l models.Lancamento
		if err := rows.Scan(&l.UUID, &l.Descricao, &l.Valor, &l.Data, &l.Categoria, &l.Ativo); err != nil {
			return nil, fmt.Errorf("erro ao escanear linha: %w", err)
		}
		lancamentos = append(lancamentos, l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro nas linhas do resultado: %w", err)
	}

	return lancamentos, nil
}

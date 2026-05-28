package repo

import "database/sql"

type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) DeleteProduct(id int) error {
	_, err := p.db.Exec("DELETE FROM products WHERE id = $1", id)
	return err
}

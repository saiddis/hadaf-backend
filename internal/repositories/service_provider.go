package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"shb/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrServiceProviderNotFound = errors.New("service provider not found")

type ServiceProviderRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewServiceProviderRepository(db *pgxpool.Pool, log *slog.Logger) *ServiceProviderRepository {
	return &ServiceProviderRepository{db: db, log: log.With("component", "ServiceProviderRepository")}
}

func (r *ServiceProviderRepository) Create(ctx context.Context, p *models.ServiceProvider) error {
	if p == nil {
		return errors.New("service provider is nil")
	}

	query := `
		INSERT INTO service_providers
		(name, type, legal_name, bank_name, bank_account, contact_phone, contact_email, address, is_verified)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		p.Name, p.Type, p.LegalName, p.BankName, p.BankAccount,
		p.ContactPhone, p.ContactEmail, p.Address, p.IsVerified,
	).Scan(&p.ID, &p.CreatedAt)

	if err != nil {
		r.log.Error("create service provider failed", "name", p.Name, "error", err)
		return fmt.Errorf("insert service provider: %w", err)
	}

	r.log.Info("service provider created", "id", p.ID, "name", p.Name)
	return nil
}

func (r *ServiceProviderRepository) GetByID(ctx context.Context, id int) (*models.ServiceProvider, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id: %d", id)
	}

	query := `SELECT id,name,type,legal_name,bank_name,bank_account,contact_phone,contact_email,address,is_verified,created_at,updated_at,deleted_at
			  FROM service_providers WHERE id=$1 AND is_deleted=false`

	var p models.ServiceProvider
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Type, &p.LegalName, &p.BankName, &p.BankAccount,
		&p.ContactPhone, &p.ContactEmail, &p.Address, &p.IsVerified,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.log.Warn("service provider not found", "id", id)
			return nil, ErrServiceProviderNotFound
		}
		r.log.Error("get service provider failed", "id", id, "error", err)
		return nil, fmt.Errorf("get service provider by id %d: %w", id, err)
	}

	return &p, nil
}

func (r *ServiceProviderRepository) List(ctx context.Context, limit, offset int) ([]*models.ServiceProvider, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id,name,type,is_verified,created_at
		FROM service_providers
		WHERE is_deleted=false
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.log.Error("list service providers failed", "limit", limit, "offset", offset, "error", err)
		return nil, fmt.Errorf("list service providers: %w", err)
	}
	defer rows.Close()

	res := make([]*models.ServiceProvider, 0, limit)
	for rows.Next() {
		var p models.ServiceProvider
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.IsVerified, &p.CreatedAt); err != nil {
			r.log.Error("scan service provider row failed", "error", err)
			return nil, fmt.Errorf("scan service provider row: %w", err)
		}
		res = append(res, &p)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("rows iteration error", "error", err)
		return nil, fmt.Errorf("iterate service provider rows: %w", err)
	}

	return res, nil
}

func (r *ServiceProviderRepository) Update(ctx context.Context, p *models.ServiceProvider) error {
	if p == nil {
		return errors.New("service provider is nil")
	}
	if p.ID <= 0 {
		return fmt.Errorf("invalid id: %d", p.ID)
	}

	query := `
		UPDATE service_providers
		SET name=$1, type=$2, legal_name=$3, bank_name=$4, bank_account=$5,
		    contact_phone=$6, contact_email=$7, address=$8, is_verified=$9, updated_at=NOW()
		WHERE id=$10 AND is_deleted=false
	`

	tag, err := r.db.Exec(ctx, query,
		p.Name, p.Type, p.LegalName, p.BankName, p.BankAccount,
		p.ContactPhone, p.ContactEmail, p.Address, p.IsVerified,
		p.ID,
	)

	if err != nil {
		r.log.Error("update service provider failed", "id", p.ID, "error", err)
		return fmt.Errorf("update service provider %d: %w", p.ID, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("update affected no rows", "id", p.ID)
		return ErrServiceProviderNotFound
	}

	r.log.Info("service provider updated", "id", p.ID)
	return nil
}

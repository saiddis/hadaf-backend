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

var ErrBeneficiaryNotFound = errors.New("beneficiary not found")

var validBeneficiaryStatuses = map[string]bool{
	"pending":  true,
	"approved": true,
	"rejected": true,
}

type BeneficiaryRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewBeneficiaryRepository(db *pgxpool.Pool, log *slog.Logger) *BeneficiaryRepository {
	return &BeneficiaryRepository{db: db, log: log.With("component", "BeneficiaryRepository")}
}

func (r *BeneficiaryRepository) Create(ctx context.Context, b *models.Beneficiary) error {
	if b == nil {
		return errors.New("beneficiary is nil")
	}
	if b.UserID <= 0 {
		return fmt.Errorf("invalid user_id: %d", b.UserID)
	}
	if b.FullName == "" {
		return errors.New("full_name is required")
	}

	query := `
		INSERT INTO beneficiaries
		(user_id, full_name, birth_date, diagnosis, city, region, contact_phone, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		b.UserID,
		b.FullName,
		b.BirthDate,
		b.Diagnosis,
		b.City,
		b.Region,
		b.ContactPhone,
		b.Status,
	).Scan(&b.ID, &b.CreatedAt)

	if err != nil {
		// note: no diagnosis/full_name in logs — PII/health data
		r.log.Error("create beneficiary failed", "user_id", b.UserID, "error", err)
		return fmt.Errorf("insert beneficiary: %w", err)
	}

	r.log.Info("beneficiary created", "id", b.ID, "user_id", b.UserID)
	return nil
}

func (r *BeneficiaryRepository) GetByID(ctx context.Context, id int) (*models.Beneficiary, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id: %d", id)
	}

	query := `
		SELECT id,user_id,full_name,birth_date,diagnosis,city,region,contact_phone,
		       status,rejection_reason,verified_by,verified_at,created_at,updated_at,deleted_at
		FROM beneficiaries
		WHERE id=$1 AND is_deleted=false
	`

	var b models.Beneficiary
	err := r.db.QueryRow(ctx, query, id).Scan(
		&b.ID,
		&b.UserID,
		&b.FullName,
		&b.BirthDate,
		&b.Diagnosis,
		&b.City,
		&b.Region,
		&b.ContactPhone,
		&b.Status,
		&b.RejectionReason,
		&b.VerifiedBy,
		&b.VerifiedAt,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.log.Warn("beneficiary not found", "id", id)
			return nil, ErrBeneficiaryNotFound
		}
		r.log.Error("get beneficiary failed", "id", id, "error", err)
		return nil, fmt.Errorf("get beneficiary by id %d: %w", id, err)
	}

	return &b, nil
}

func (r *BeneficiaryRepository) GetByUserID(ctx context.Context, userID int) ([]*models.Beneficiary, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user_id: %d", userID)
	}

	query := `
		SELECT id,user_id,full_name,birth_date,diagnosis,status,created_at
		FROM beneficiaries
		WHERE user_id=$1 AND is_deleted=false
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("list beneficiaries by user failed", "user_id", userID, "error", err)
		return nil, fmt.Errorf("get beneficiaries by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var res []*models.Beneficiary
	for rows.Next() {
		var b models.Beneficiary
		if err := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.FullName,
			&b.BirthDate,
			&b.Diagnosis,
			&b.Status,
			&b.CreatedAt,
		); err != nil {
			r.log.Error("scan beneficiary row failed", "user_id", userID, "error", err)
			return nil, fmt.Errorf("scan beneficiary row: %w", err)
		}
		res = append(res, &b)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("rows iteration error", "user_id", userID, "error", err)
		return nil, fmt.Errorf("iterate beneficiary rows: %w", err)
	}

	return res, nil
}

func (r *BeneficiaryRepository) UpdateStatus(
	ctx context.Context,
	id int,
	status string,
	reason *string,
	verifiedBy *int,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid id: %d", id)
	}
	if !validBeneficiaryStatuses[status] {
		return fmt.Errorf("invalid status: %q", status)
	}
	if status == "rejected" && (reason == nil || *reason == "") {
		return errors.New("rejection_reason is required when status is rejected")
	}

	query := `
		UPDATE beneficiaries
		SET status=$1,
		    rejection_reason=$2,
		    verified_by=$3,
		    verified_at=NOW(),
		    updated_at=NOW()
		WHERE id=$4 AND is_deleted=false
	`

	tag, err := r.db.Exec(ctx, query,
		status,
		reason,
		verifiedBy,
		id,
	)

	if err != nil {
		r.log.Error("update beneficiary status failed", "id", id, "status", status, "error", err)
		return fmt.Errorf("update beneficiary status %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("update status affected no rows", "id", id)
		return ErrBeneficiaryNotFound
	}

	r.log.Info("beneficiary status updated", "id", id, "status", status, "verified_by", verifiedBy)
	return nil
}

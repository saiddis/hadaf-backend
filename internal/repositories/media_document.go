package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"shb/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMediaNotFound    = errors.New("media not found")
	ErrDocumentNotFound = errors.New("verification document not found")
)

var validDocumentStatuses = map[string]bool{
	"pending":  true,
	"approved": true,
	"rejected": true,
}

type MediaDocumentRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewMediaDocumentRepository(db *pgxpool.Pool, log *slog.Logger) *MediaDocumentRepository {
	return &MediaDocumentRepository{db: db, log: log.With("component", "MediaDocumentRepository")}
}

//
// =========================
// BENEFICIARY MEDIA
// =========================
//

func (r *MediaDocumentRepository) CreateMedia(ctx context.Context, m *models.BeneficiaryMedia) error {
	if m == nil {
		return errors.New("beneficiary media is nil")
	}
	if m.BeneficiaryID <= 0 {
		return fmt.Errorf("invalid beneficiary_id: %d", m.BeneficiaryID)
	}
	if m.FilePath == "" || m.FileHash == "" {
		return errors.New("file_path and file_hash are required")
	}

	query := `
		INSERT INTO beneficiary_media
		(beneficiary_id, file_path, file_hash, original_name, mime_type)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, uploaded_at
	`

	err := r.db.QueryRow(ctx, query,
		m.BeneficiaryID,
		m.FilePath,
		m.FileHash,
		m.OriginalName,
		m.MimeType,
	).Scan(&m.ID, &m.UploadedAt)

	if err != nil {
		r.log.Error("create beneficiary media failed", "beneficiary_id", m.BeneficiaryID, "error", err)
		return fmt.Errorf("insert beneficiary media: %w", err)
	}

	r.log.Info("beneficiary media created", "id", m.ID, "beneficiary_id", m.BeneficiaryID)
	return nil
}

func (r *MediaDocumentRepository) ApproveMedia(ctx context.Context, id int, adminID int) error {
	if id <= 0 {
		return fmt.Errorf("invalid id: %d", id)
	}
	if adminID <= 0 {
		return fmt.Errorf("invalid admin_id: %d", adminID)
	}

	query := `
		UPDATE beneficiary_media
		SET is_approved=true,
		    reviewed_by=$1,
		    reviewed_at=NOW()
		WHERE id=$2 AND is_deleted=false
	`

	tag, err := r.db.Exec(ctx, query, adminID, id)
	if err != nil {
		r.log.Error("approve media failed", "id", id, "admin_id", adminID, "error", err)
		return fmt.Errorf("approve media %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("approve media affected no rows", "id", id)
		return ErrMediaNotFound
	}

	r.log.Info("media approved", "id", id, "admin_id", adminID)
	return nil
}

//
// =========================
// VERIFICATION DOCUMENTS
// =========================
//

func (r *MediaDocumentRepository) CreateDocument(ctx context.Context, d *models.VerificationDocument) error {
	if d == nil {
		return errors.New("verification document is nil")
	}
	if d.BeneficiaryID == nil || *d.BeneficiaryID <= 0 {
		return fmt.Errorf("invalid beneficiary_id: %v", d.BeneficiaryID)
	}
	if d.FilePath == "" || d.FileHash == "" {
		return errors.New("file_path and file_hash are required")
	}
	if d.FileSize == nil || *d.FileSize <= 0 {
		return fmt.Errorf("invalid file_size: %v", d.FileSize)
	}

	query := `
		INSERT INTO verification_documents
		(beneficiary_id, campaign_id, document_type, file_path, file_hash, file_size, original_name, mime_type, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, uploaded_at
	`

	err := r.db.QueryRow(ctx, query,
		d.BeneficiaryID,
		d.CampaignID,
		d.DocumentType,
		d.FilePath,
		d.FileHash,
		d.FileSize,
		d.OriginalName,
		d.MimeType,
		d.Status,
	).Scan(&d.ID, &d.UploadedAt)

	if err != nil {
		r.log.Error("create verification document failed",
			"beneficiary_id", d.BeneficiaryID, "campaign_id", d.CampaignID, "error", err)
		return fmt.Errorf("insert verification document: %w", err)
	}

	r.log.Info("verification document created", "id", d.ID, "beneficiary_id", d.BeneficiaryID)
	return nil
}

func (r *MediaDocumentRepository) UpdateDocumentStatus(
	ctx context.Context,
	id int,
	status string,
	rejectionNote *string,
	reviewedBy *int,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid id: %d", id)
	}
	if !validDocumentStatuses[status] {
		return fmt.Errorf("invalid status: %q", status)
	}
	if status == "rejected" && (rejectionNote == nil || *rejectionNote == "") {
		return errors.New("rejection_note is required when status is rejected")
	}

	query := `
		UPDATE verification_documents
		SET status=$1,
		    rejection_note=$2,
		    reviewed_by=$3,
		    reviewed_at=NOW()
		WHERE id=$4 AND is_deleted=false
	`

	tag, err := r.db.Exec(ctx, query,
		status,
		rejectionNote,
		reviewedBy,
		id,
	)

	if err != nil {
		r.log.Error("update document status failed", "id", id, "status", status, "error", err)
		return fmt.Errorf("update document status %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("update document status affected no rows", "id", id)
		return ErrDocumentNotFound
	}

	r.log.Info("document status updated", "id", id, "status", status, "reviewed_by", reviewedBy)
	return nil
}

//
// =========================
// ANTI-FRAUD CHECK
// =========================
//

func (r *MediaDocumentRepository) CheckFileHashExists(ctx context.Context, hash string) (bool, error) {
	if hash == "" {
		return false, errors.New("hash is required")
	}

	query := `
		SELECT EXISTS (
			SELECT 1 FROM beneficiary_media WHERE file_hash=$1 AND is_deleted=false
			UNION
			SELECT 1 FROM verification_documents WHERE file_hash=$1 AND is_deleted=false
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, hash).Scan(&exists); err != nil {
		r.log.Error("check file hash failed", "hash", hash, "error", err)
		return false, fmt.Errorf("check file hash exists: %w", err)
	}

	if exists {
		r.log.Warn("duplicate file hash detected", "hash", hash)
	}

	return exists, nil
}

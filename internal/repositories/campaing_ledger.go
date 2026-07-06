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

var (
	ErrCampaignNotFound  = errors.New("campaign not found")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInvalidCampaignID = errors.New("invalid campaign id")
)

var validCampaignStatuses = map[string]bool{
	"draft":     true,
	"active":    true,
	"completed": true,
	"cancelled": true,
}

type CampaignLedgerRepository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewCampaignLedgerRepository(db *pgxpool.Pool, log *slog.Logger) *CampaignLedgerRepository {
	return &CampaignLedgerRepository{db: db, log: log.With("component", "CampaignLedgerRepository")}
}

//
// =========================
// CAMPAIGNS
// =========================
//

func (r *CampaignLedgerRepository) CreateCampaign(ctx context.Context, c *models.MedicalCampaign) error {
	if c == nil {
		return errors.New("campaign is nil")
	}
	if c.BeneficiaryID <= 0 {
		return fmt.Errorf("invalid beneficiary_id: %d", c.BeneficiaryID)
	}
	if c.ProviderID == nil || *c.ProviderID <= 0 {
		return fmt.Errorf("invalid provider_id: %v", c.ProviderID)
	}
	if c.TargetAmount <= 0 {
		return fmt.Errorf("%w: target_amount must be positive", ErrInvalidAmount)
	}

	query := `
		INSERT INTO medical_campaigns
		(beneficiary_id, provider_id, title, description, target_amount, collected_amount, currency, invoice_number, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		c.BeneficiaryID,
		c.ProviderID,
		c.Title,
		c.Description,
		c.TargetAmount,
		c.CollectedAmount,
		c.Currency,
		c.InvoiceNumber,
		c.Status,
	).Scan(&c.ID, &c.CreatedAt)

	if err != nil {
		r.log.Error("create campaign failed",
			"beneficiary_id", c.BeneficiaryID, "provider_id", c.ProviderID, "error", err)
		return fmt.Errorf("insert campaign: %w", err)
	}

	r.log.Info("campaign created", "id", c.ID, "beneficiary_id", c.BeneficiaryID, "target_amount", c.TargetAmount)
	return nil
}

func (r *CampaignLedgerRepository) GetByID(ctx context.Context, id int) (*models.MedicalCampaign, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidCampaignID, id)
	}

	query := `
		SELECT id,beneficiary_id,provider_id,title,description,target_amount,
		       collected_amount,currency,invoice_number,status,
		       deadline,completed_at,cancelled_reason,reviewed_by,
		       reviewed_at,created_at,updated_at,deleted_at
		FROM medical_campaigns
		WHERE id=$1 AND is_deleted=false
	`

	var c models.MedicalCampaign
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.BeneficiaryID,
		&c.ProviderID,
		&c.Title,
		&c.Description,
		&c.TargetAmount,
		&c.CollectedAmount,
		&c.Currency,
		&c.InvoiceNumber,
		&c.Status,
		&c.Deadline,
		&c.CompletedAt,
		&c.CancelledReason,
		&c.ReviewedBy,
		&c.ReviewedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.log.Warn("campaign not found", "id", id)
			return nil, ErrCampaignNotFound
		}
		r.log.Error("get campaign failed", "id", id, "error", err)
		return nil, fmt.Errorf("get campaign by id %d: %w", id, err)
	}

	return &c, nil
}

func (r *CampaignLedgerRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	if id <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidCampaignID, id)
	}
	if !validCampaignStatuses[status] {
		return fmt.Errorf("invalid status: %q", status)
	}

	query := `
		UPDATE medical_campaigns
		SET status=$1,
		    updated_at=NOW()
		WHERE id=$2 AND is_deleted=false
	`

	tag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		r.log.Error("update campaign status failed", "id", id, "status", status, "error", err)
		return fmt.Errorf("update campaign status %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("update campaign status affected no rows", "id", id)
		return ErrCampaignNotFound
	}

	r.log.Info("campaign status updated", "id", id, "status", status)
	return nil
}

//
// =========================
// ATOMIC MONEY UPDATE (CRITICAL)
// =========================
//

// incrementCollectedAmount is unexported: it must only run inside a transaction
// alongside a ledger entry write (see RecordDonation), otherwise collected_amount
// and the ledger can drift apart.
func (r *CampaignLedgerRepository) incrementCollectedAmount(
	ctx context.Context,
	tx pgx.Tx,
	campaignID int,
	amount float64,
) error {
	if campaignID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidCampaignID, campaignID)
	}
	if amount == 0 {
		return fmt.Errorf("%w: got 0", ErrInvalidAmount)
	}

	query := `
		UPDATE medical_campaigns
		SET collected_amount = collected_amount + $1,
		    updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
	`

	tag, err := tx.Exec(ctx, query, amount, campaignID)
	if err != nil {
		r.log.Error("increment collected amount failed",
			"campaign_id", campaignID, "amount", amount, "error", err)
		return fmt.Errorf("increment collected amount for campaign %d: %w", campaignID, err)
	}

	if tag.RowsAffected() == 0 {
		r.log.Warn("increment collected amount affected no rows", "campaign_id", campaignID)
		return ErrCampaignNotFound
	}

	return nil
}

//
// =========================
// LEDGER
// =========================
//

// RecordDonation writes a ledger entry and updates the campaign's collected_amount
// atomically in a single transaction. Use this instead of calling CreateLedgerEntry
// and incrementCollectedAmount separately — doing them independently risks a ledger
// entry existing with no matching balance update, or vice versa, if either fails.
func (r *CampaignLedgerRepository) RecordDonation(ctx context.Context, e *models.LedgerEntry) error {
	if e == nil {
		return errors.New("ledger entry is nil")
	}
	if e.CampaignID == nil || *e.CampaignID <= 0 {
		return fmt.Errorf("%w: %v", ErrInvalidCampaignID, e.CampaignID)
	}
	if e.Amount <= 0 {
		return fmt.Errorf("%w: %f", ErrInvalidAmount, e.Amount)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.log.Error("begin transaction failed", "campaign_id", e.CampaignID, "error", err)
		return fmt.Errorf("begin donation transaction: %w", err)
	}
	defer func() {
		// no-op if already committed
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			r.log.Error("rollback failed", "campaign_id", e.CampaignID, "error", rbErr)
		}
	}()

	query := `
		INSERT INTO ledger
		(campaign_id, donor_user_id, type, amount, currency, payment_ref, description, is_anonymous, donor_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at
	`

	err = tx.QueryRow(ctx, query,
		*e.CampaignID,
		e.DonorUserID,
		e.Type,
		e.Amount,
		e.Currency,
		e.PaymentRef,
		e.Description,
		e.IsAnonymous,
		e.DonorMessage,
	).Scan(&e.ID, &e.CreatedAt)

	if err != nil {
		r.log.Error("create ledger entry failed", "campaign_id", e.CampaignID, "error", err)
		return fmt.Errorf("insert ledger entry: %w", err)
	}

	if err := r.incrementCollectedAmount(ctx, tx, *e.CampaignID, e.Amount); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		r.log.Error("commit donation transaction failed", "campaign_id", e.CampaignID, "error", err)
		return fmt.Errorf("commit donation transaction: %w", err)
	}

	r.log.Info("donation recorded", "ledger_id", e.ID, "campaign_id", e.CampaignID, "amount", e.Amount)
	return nil
}

func (r *CampaignLedgerRepository) CreateLedgerEntry(ctx context.Context, e *models.LedgerEntry) error {
	if e == nil {
		return errors.New("ledger entry is nil")
	}
	if e.CampaignID == nil || *e.CampaignID <= 0 {
		return fmt.Errorf("%w: %v", ErrInvalidCampaignID, e.CampaignID)
	}
	if e.Amount <= 0 {
		return fmt.Errorf("%w: %f", ErrInvalidAmount, e.Amount)
	}

	query := `
		INSERT INTO ledger
		(campaign_id, donor_user_id, type, amount, currency, payment_ref, description, is_anonymous, donor_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		*e.CampaignID,
		e.DonorUserID,
		e.Type,
		e.Amount,
		e.Currency,
		e.PaymentRef,
		e.Description,
		e.IsAnonymous,
		e.DonorMessage,
	).Scan(&e.ID, &e.CreatedAt)

	if err != nil {
		r.log.Error("create ledger entry failed", "campaign_id", e.CampaignID, "error", err)
		return fmt.Errorf("insert ledger entry: %w", err)
	}

	r.log.Info("ledger entry created", "id", e.ID, "campaign_id", e.CampaignID, "type", e.Type)
	return nil
}

func (r *CampaignLedgerRepository) GetTotalDonationsByCampaign(
	ctx context.Context,
	campaignID int,
) (float64, error) {
	if campaignID <= 0 {
		return 0, fmt.Errorf("%w: %d", ErrInvalidCampaignID, campaignID)
	}

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM ledger
		WHERE campaign_id = $1 AND type = 'donation'
	`

	var total float64
	if err := r.db.QueryRow(ctx, query, campaignID).Scan(&total); err != nil {
		r.log.Error("get total donations failed", "campaign_id", campaignID, "error", err)
		return 0, fmt.Errorf("get total donations for campaign %d: %w", campaignID, err)
	}

	return total, nil
}

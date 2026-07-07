-- Поставщики услуг (клиники, аптеки, реаб.центры)
CREATE TABLE IF NOT EXISTS service_providers (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(300) NOT NULL,
    type          VARCHAR(50) NOT NULL,       -- clinic | pharmacy | rehab_center
    legal_name    VARCHAR(500),
    bank_name     VARCHAR(200),
    bank_account  VARCHAR(100),
    contact_phone VARCHAR(50),
    contact_email VARCHAR(150),
    address       TEXT,
    is_verified   BOOLEAN DEFAULT FALSE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ,
    is_deleted    BOOLEAN DEFAULT FALSE,
    deleted_at    TIMESTAMPTZ
);

-- Благополучатели (только дети с ДЦП до 12 лет на MVP)
CREATE TABLE IF NOT EXISTS beneficiaries (
    id              SERIAL PRIMARY KEY,
    user_id         INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name       VARCHAR(255) NOT NULL,
    birth_date      DATE NOT NULL,
    diagnosis       VARCHAR(50) NOT NULL DEFAULT 'cerebral_palsy',
    city            VARCHAR(100),
    region          VARCHAR(100),
    contact_phone   VARCHAR(50),
    status          VARCHAR(30) DEFAULT 'pending',
    rejection_reason TEXT,
    verified_by     INT REFERENCES users(id),
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT chk_diagnosis CHECK (diagnosis IN ('cerebral_palsy'))
);
CREATE INDEX idx_beneficiaries_user ON beneficiaries(user_id);
CREATE INDEX idx_beneficiaries_status ON beneficiaries(status);

-- Медиа-файлы (фото подопечного) с модерацией
CREATE TABLE IF NOT EXISTS beneficiary_media (
    id              SERIAL PRIMARY KEY,
    beneficiary_id  INT NOT NULL REFERENCES beneficiaries(id) ON DELETE CASCADE,
    file_path       VARCHAR(500) NOT NULL,
    file_hash       VARCHAR(128) NOT NULL,
    original_name   VARCHAR(300),
    mime_type       VARCHAR(100),
    is_approved     BOOLEAN DEFAULT FALSE,
    reviewed_by     INT REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    uploaded_at     TIMESTAMPTZ DEFAULT NOW(),
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_media_hash ON beneficiary_media(file_hash) WHERE is_deleted = FALSE;

-- Верификационные документы
CREATE TABLE IF NOT EXISTS verification_documents (
    id              SERIAL PRIMARY KEY,
    beneficiary_id  INT REFERENCES beneficiaries(id) ON DELETE CASCADE,
    campaign_id     INT,  -- FK добавим после создания campaigns
    document_type   VARCHAR(50) NOT NULL,
    file_path       VARCHAR(500) NOT NULL,
    file_hash       VARCHAR(128) NOT NULL,
    file_size       BIGINT,
    original_name   VARCHAR(300),
    mime_type       VARCHAR(100),
    status          VARCHAR(30) DEFAULT 'pending',
    rejection_note  TEXT,
    reviewed_by     INT REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    uploaded_at     TIMESTAMPTZ DEFAULT NOW(),
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_doc_hash_type ON verification_documents(file_hash, document_type) WHERE is_deleted = FALSE;

-- Кампании сбора средств
CREATE TABLE IF NOT EXISTS medical_campaigns (
    id                SERIAL PRIMARY KEY,
    beneficiary_id    INT NOT NULL REFERENCES beneficiaries(id) ON DELETE CASCADE,
    provider_id       INT REFERENCES service_providers(id),
    title             VARCHAR(500) NOT NULL,
    description       TEXT,
    target_amount     DECIMAL(12,2) NOT NULL,
    collected_amount  DECIMAL(12,2) DEFAULT 0,
    currency          VARCHAR(10) DEFAULT 'TJS',
    invoice_number    VARCHAR(100),
    status            VARCHAR(30) DEFAULT 'draft',
    deadline          TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    cancelled_reason  TEXT,
    reviewed_by       INT REFERENCES users(id),
    reviewed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ,
    is_deleted        BOOLEAN DEFAULT FALSE,
    deleted_at        TIMESTAMPTZ
);
ALTER TABLE verification_documents ADD CONSTRAINT fk_verdoc_campaign
    FOREIGN KEY (campaign_id) REFERENCES medical_campaigns(id) ON DELETE CASCADE;
CREATE INDEX idx_campaigns_status ON medical_campaigns(status);

-- Ledger (транзакции / внутренний учёт)
CREATE TABLE IF NOT EXISTS ledger (
    id              SERIAL PRIMARY KEY,
    campaign_id     INT REFERENCES medical_campaigns(id),
    donor_user_id   INT REFERENCES users(id),
    type            VARCHAR(30) NOT NULL,    -- donation | payment_to_provider | overflow_to_general | refund
    amount          DECIMAL(12,2) NOT NULL,
    currency        VARCHAR(10) DEFAULT 'TJS',
    payment_ref     VARCHAR(200),
    description     TEXT,
    is_anonymous    BOOLEAN DEFAULT FALSE,
    donor_message   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ledger_campaign ON ledger(campaign_id);
CREATE INDEX idx_ledger_type ON ledger(type);

-- Отчёты о расходах (оплата клинике)
CREATE TABLE IF NOT EXISTS campaign_expenditures (
    id              SERIAL PRIMARY KEY,
    campaign_id     INT NOT NULL REFERENCES medical_campaigns(id),
    provider_id     INT REFERENCES service_providers(id),
    amount          DECIMAL(12,2) NOT NULL,
    description     TEXT NOT NULL,
    receipt_path    VARCHAR(500),
    invoice_path    VARCHAR(500),
    paid_at         TIMESTAMPTZ,
    created_by      INT REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ
);

-- Триггеры
CREATE TRIGGER trg_updated_at BEFORE UPDATE ON service_providers FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_updated_at BEFORE UPDATE ON beneficiaries FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_updated_at BEFORE UPDATE ON medical_campaigns FOR EACH ROW EXECUTE FUNCTION set_updated_at();

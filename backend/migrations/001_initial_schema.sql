-- Migration: 001_initial_schema
-- Description: Initial database schema for Web3AirdropOS
-- Created: 2026-01-08

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- USERS & AUTH
-- ============================================================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    settings JSONB DEFAULT '{}',
    open_ai_key TEXT, -- Encrypted
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) UNIQUE NOT NULL,
    refresh_token VARCHAR(500) UNIQUE NOT NULL,
    user_agent VARCHAR(500),
    ip_address VARCHAR(50),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    family_id UUID NOT NULL, -- Token family for rotation detection
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(50),
    user_agent VARCHAR(500)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- ============================================================================
-- WALLETS
-- ============================================================================

CREATE TABLE IF NOT EXISTS wallet_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wallet_tags_user_id ON wallet_tags(user_id);

CREATE TABLE IF NOT EXISTS wallet_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_wallet_groups_user_id ON wallet_groups(user_id);

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100),
    address VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL, -- evm, solana, cosmos, etc.
    encrypted_key TEXT NOT NULL, -- AES-256-GCM encrypted private key
    public_key VARCHAR(200),
    is_imported BOOLEAN DEFAULT false,
    balance VARCHAR(50) DEFAULT '0',
    last_synced_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_wallets_address ON wallets(address);
CREATE INDEX idx_wallets_type ON wallets(type);
CREATE INDEX idx_wallets_deleted_at ON wallets(deleted_at);

CREATE TABLE IF NOT EXISTS wallet_tags_wallets (
    wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE,
    wallet_tag_id UUID REFERENCES wallet_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (wallet_id, wallet_tag_id)
);

CREATE TABLE IF NOT EXISTS wallet_groups_wallets (
    wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE,
    wallet_group_id UUID REFERENCES wallet_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (wallet_id, wallet_group_id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(100) UNIQUE,
    from_address VARCHAR(100) NOT NULL,
    to_address VARCHAR(100),
    value VARCHAR(100),
    gas_used BIGINT,
    gas_price VARCHAR(100),
    status VARCHAR(20) NOT NULL, -- pending, confirmed, failed
    block_number BIGINT,
    data TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMPTZ
);

CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- ============================================================================
-- PLATFORM ACCOUNTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS proxies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(20) NOT NULL, -- http, https, socks5
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    username VARCHAR(100),
    password_encrypted TEXT, -- Encrypted password
    is_active BOOLEAN DEFAULT true,
    last_checked_at TIMESTAMPTZ,
    last_latency_ms INTEGER,
    fail_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_proxies_user_id ON proxies(user_id);
CREATE INDEX idx_proxies_is_active ON proxies(is_active);

CREATE TABLE IF NOT EXISTS platform_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID REFERENCES wallets(id) ON DELETE SET NULL,
    proxy_id UUID REFERENCES proxies(id) ON DELETE SET NULL,
    platform VARCHAR(30) NOT NULL, -- farcaster, telegram, twitter, discord
    platform_user_id VARCHAR(200),
    username VARCHAR(100),
    display_name VARCHAR(200),
    avatar_url VARCHAR(500),
    profile_url VARCHAR(500),
    credentials_encrypted TEXT, -- Encrypted API keys/tokens
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    last_synced_at TIMESTAMPTZ,
    follower_count INTEGER DEFAULT 0,
    following_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_platform_accounts_user_id ON platform_accounts(user_id);
CREATE INDEX idx_platform_accounts_platform ON platform_accounts(platform);
CREATE INDEX idx_platform_accounts_wallet_id ON platform_accounts(wallet_id);
CREATE INDEX idx_platform_accounts_username ON platform_accounts(username);

CREATE TABLE IF NOT EXISTS account_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES platform_accounts(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    target_id VARCHAR(200),
    target_url VARCHAR(500),
    result VARCHAR(20) NOT NULL, -- success, failed, pending
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_account_activities_account_id ON account_activities(account_id);
CREATE INDEX idx_account_activities_action ON account_activities(action);
CREATE INDEX idx_account_activities_created_at ON account_activities(created_at);

-- ============================================================================
-- CAMPAIGNS & TASKS
-- ============================================================================

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- airdrop, galxe, zealy, layer3, custom, farcaster
    url VARCHAR(500),
    image_url VARCHAR(500),
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    deadline TIMESTAMPTZ,
    status VARCHAR(30) DEFAULT 'active', -- active, paused, completed, expired
    priority INTEGER DEFAULT 0,
    estimated_reward VARCHAR(100),
    reward_type VARCHAR(50), -- token, nft, points, unknown
    total_tasks INTEGER DEFAULT 0,
    completed_tasks INTEGER DEFAULT 0,
    progress_percent DECIMAL(5,2) DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_campaigns_user_id ON campaigns(user_id);
CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_type ON campaigns(type);
CREATE INDEX idx_campaigns_deadline ON campaigns(deadline);
CREATE INDEX idx_campaigns_deleted_at ON campaigns(deleted_at);

CREATE TABLE IF NOT EXISTS campaign_wallet_groups (
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    wallet_group_id UUID REFERENCES wallet_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (campaign_id, wallet_group_id)
);

CREATE TABLE IF NOT EXISTS campaign_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- wallet_connect, transaction, claim, follow, join, post, reply, like, recast, verify, quiz, custom
    target_url VARCHAR(500),
    target_platform VARCHAR(50),
    target_account VARCHAR(200),
    required_action TEXT,
    is_automatable BOOLEAN DEFAULT true,
    automation_script TEXT,
    requires_manual BOOLEAN DEFAULT false,
    task_order INTEGER DEFAULT 0,
    depends_on UUID REFERENCES campaign_tasks(id) ON DELETE SET NULL,
    points INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_campaign_tasks_campaign_id ON campaign_tasks(campaign_id);
CREATE INDEX idx_campaign_tasks_type ON campaign_tasks(type);
CREATE INDEX idx_campaign_tasks_order ON campaign_tasks(task_order);

-- Task execution status: PENDING -> RUNNING -> (MANUAL_REQUIRED) -> DONE/FAILED
CREATE TYPE task_status AS ENUM ('PENDING', 'RUNNING', 'MANUAL_REQUIRED', 'DONE', 'FAILED', 'SKIPPED');

CREATE TABLE IF NOT EXISTS task_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES campaign_tasks(id) ON DELETE CASCADE,
    wallet_id UUID REFERENCES wallets(id) ON DELETE SET NULL,
    account_id UUID REFERENCES platform_accounts(id) ON DELETE SET NULL,
    
    -- Status with enum
    status task_status NOT NULL DEFAULT 'PENDING',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Idempotency - prevents duplicate executions
    idempotency_key VARCHAR(200) UNIQUE NOT NULL,
    
    -- Proof of completion
    proof_type VARCHAR(50), -- post_url, tx_hash, cast_hash, screenshot, api_response
    proof_value VARCHAR(500),
    proof_data JSONB,
    screenshot_path VARCHAR(500),
    
    -- Result data
    transaction_hash VARCHAR(100),
    post_id VARCHAR(200),
    post_url VARCHAR(500),
    result_data JSONB,
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Browser session reference
    browser_session_id UUID,
    
    -- Retry info
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    
    -- Audit link
    audit_log_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_task_executions_task_id ON task_executions(task_id);
CREATE INDEX idx_task_executions_wallet_id ON task_executions(wallet_id);
CREATE INDEX idx_task_executions_account_id ON task_executions(account_id);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_idempotency_key ON task_executions(idempotency_key);
CREATE INDEX idx_task_executions_created_at ON task_executions(created_at);

-- ============================================================================
-- AUTOMATION & JOBS
-- ============================================================================

CREATE TABLE IF NOT EXISTS automation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- scheduled_post, campaign_task, balance_sync, platform_sync, engagement, content_generate, bulk_execute
    name VARCHAR(200),
    description TEXT,
    cron_expression VARCHAR(100),
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    status VARCHAR(30) DEFAULT 'idle', -- idle, running, paused, failed
    config JSONB DEFAULT '{}',
    wallet_ids JSONB,
    account_ids JSONB,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    total_runs INTEGER DEFAULT 0,
    success_runs INTEGER DEFAULT 0,
    failed_runs INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_automation_jobs_user_id ON automation_jobs(user_id);
CREATE INDEX idx_automation_jobs_type ON automation_jobs(type);
CREATE INDEX idx_automation_jobs_status ON automation_jobs(status);
CREATE INDEX idx_automation_jobs_next_run_at ON automation_jobs(next_run_at);

CREATE TABLE IF NOT EXISTS job_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL, -- info, warn, error, debug
    message TEXT NOT NULL,
    details JSONB,
    wallet_id UUID,
    account_id UUID,
    task_id UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_job_logs_job_id ON job_logs(job_id);
CREATE INDEX idx_job_logs_level ON job_logs(level);
CREATE INDEX idx_job_logs_created_at ON job_logs(created_at);

-- ============================================================================
-- CONTENT & SCHEDULING
-- ============================================================================

CREATE TABLE IF NOT EXISTS content_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(30),
    type VARCHAR(30), -- post, reply, thread
    content TEXT NOT NULL,
    media_urls JSONB,
    prompt TEXT,
    ai_model VARCHAR(50),
    tone VARCHAR(30),
    status VARCHAR(30) DEFAULT 'drafted', -- drafted, awaiting_approval, approved, scheduled, published, failed
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    rejected_at TIMESTAMPTZ,
    rejection_reason TEXT,
    published_at TIMESTAMPTZ,
    published_post_id VARCHAR(200),
    published_url VARCHAR(500),
    target_account_id UUID REFERENCES platform_accounts(id) ON DELETE SET NULL,
    predicted_engagement JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_content_drafts_user_id ON content_drafts(user_id);
CREATE INDEX idx_content_drafts_status ON content_drafts(status);

CREATE TABLE IF NOT EXISTS scheduled_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES platform_accounts(id) ON DELETE CASCADE,
    draft_id UUID REFERENCES content_drafts(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    media_urls JSONB,
    platform VARCHAR(50) NOT NULL,
    reply_to_id VARCHAR(200),
    reply_to_url VARCHAR(500),
    scheduled_for TIMESTAMPTZ NOT NULL,
    scheduled_at TIMESTAMPTZ,
    timezone VARCHAR(50),
    status VARCHAR(30) DEFAULT 'pending', -- pending, posted, failed, cancelled
    posted_at TIMESTAMPTZ,
    post_id VARCHAR(200),
    post_url VARCHAR(500),
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_scheduled_posts_user_id ON scheduled_posts(user_id);
CREATE INDEX idx_scheduled_posts_account_id ON scheduled_posts(account_id);
CREATE INDEX idx_scheduled_posts_scheduled_for ON scheduled_posts(scheduled_for);
CREATE INDEX idx_scheduled_posts_status ON scheduled_posts(status);

-- ============================================================================
-- BROWSER SESSIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS browser_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100),
    user_agent VARCHAR(500),
    viewport_width INTEGER DEFAULT 1920,
    viewport_height INTEGER DEFAULT 1080,
    proxy_id UUID REFERENCES proxies(id) ON DELETE SET NULL,
    cookies_encrypted TEXT,
    local_storage_encrypted TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_browser_profiles_user_id ON browser_profiles(user_id);

CREATE TABLE IF NOT EXISTS browser_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES browser_profiles(id) ON DELETE SET NULL,
    container_id VARCHAR(100),
    ws_endpoint VARCHAR(500),
    status VARCHAR(30), -- starting, active, idle, closed, error
    current_url VARCHAR(1000),
    screenshot_path VARCHAR(500),
    started_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_browser_sessions_user_id ON browser_sessions(user_id);
CREATE INDEX idx_browser_sessions_status ON browser_sessions(status);
CREATE INDEX idx_browser_sessions_profile_id ON browser_sessions(profile_id);

CREATE TABLE IF NOT EXISTS browser_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES browser_sessions(id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL, -- navigate, click, type, scroll, wait, screenshot, execute_script
    target VARCHAR(500),
    value TEXT,
    status VARCHAR(30), -- pending, running, completed, failed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    result JSONB,
    error_message TEXT,
    screenshot_before VARCHAR(500),
    screenshot_after VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_browser_actions_session_id ON browser_actions(session_id);
CREATE INDEX idx_browser_actions_action_type ON browser_actions(action_type);

-- ============================================================================
-- AUDIT LOGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Who
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES platform_accounts(id) ON DELETE SET NULL,
    wallet_id UUID REFERENCES wallets(id) ON DELETE SET NULL,
    profile_id UUID,
    
    -- What
    action VARCHAR(50) NOT NULL, -- follow, unfollow, like, unlike, repost, post, reply, quote, delete, transaction, token_approval, swap, bridge, mint, claim, generate, approve, reject, schedule, publish, login, account_link, wallet_create, wallet_import, task_start, task_complete, task_fail, job_run, browser_action
    platform VARCHAR(30),
    target_type VARCHAR(50),
    target_id VARCHAR(200),
    
    -- Context
    task_id UUID,
    execution_id UUID,
    job_id UUID,
    campaign_id UUID,
    session_id UUID,
    
    -- Result
    result VARCHAR(20) NOT NULL, -- success, failed, pending, skipped
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Proof
    proof_type VARCHAR(50),
    proof_value VARCHAR(500),
    proof_data JSONB,
    
    -- Request/Response for debugging
    request_data JSONB,
    response_data JSONB,
    
    -- Metadata
    ip_address VARCHAR(50),
    user_agent VARCHAR(300),
    duration_ms BIGINT,
    
    -- Idempotency
    idempotency_key VARCHAR(200) UNIQUE,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_account_id ON audit_logs(account_id);
CREATE INDEX idx_audit_logs_wallet_id ON audit_logs(wallet_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_platform ON audit_logs(platform);
CREATE INDEX idx_audit_logs_result ON audit_logs(result);
CREATE INDEX idx_audit_logs_task_id ON audit_logs(task_id);
CREATE INDEX idx_audit_logs_execution_id ON audit_logs(execution_id);
CREATE INDEX idx_audit_logs_campaign_id ON audit_logs(campaign_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_idempotency_key ON audit_logs(idempotency_key);

-- Partitioning for audit_logs (by month) - optional for scale
-- Can be enabled later with: CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- ============================================================================
-- SECRETS VAULT
-- ============================================================================

CREATE TABLE IF NOT EXISTS secrets_vault (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_type VARCHAR(50) NOT NULL, -- api_key, token, password, private_key, certificate
    encrypted_value TEXT NOT NULL, -- AES-256-GCM encrypted
    iv VARCHAR(32) NOT NULL, -- Initialization vector for AES-GCM
    metadata JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    last_accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    
    UNIQUE(user_id, name)
);

CREATE INDEX idx_secrets_vault_user_id ON secrets_vault(user_id);
CREATE INDEX idx_secrets_vault_key_type ON secrets_vault(key_type);
CREATE INDEX idx_secrets_vault_name ON secrets_vault(user_id, name);

-- ============================================================================
-- SCHEMA MIGRATIONS TRACKING
-- ============================================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64),
    execution_time_ms INTEGER
);

-- Record this migration
INSERT INTO schema_migrations (version, name, checksum) 
VALUES ('001', 'initial_schema', 'auto-generated')
ON CONFLICT (version) DO NOTHING;

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers to all relevant tables
DO $$
DECLARE
    t text;
BEGIN
    FOR t IN 
        SELECT table_name 
        FROM information_schema.columns 
        WHERE column_name = 'updated_at' 
        AND table_schema = 'public'
    LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
            CREATE TRIGGER update_%s_updated_at
                BEFORE UPDATE ON %s
                FOR EACH ROW
                EXECUTE FUNCTION update_updated_at_column();
        ', t, t, t, t);
    END LOOP;
END;
$$;

-- Function to generate idempotency keys
CREATE OR REPLACE FUNCTION generate_idempotency_key(
    task_id UUID,
    account_id UUID DEFAULT NULL,
    wallet_id UUID DEFAULT NULL,
    date_suffix DATE DEFAULT CURRENT_DATE
) RETURNS VARCHAR(200) AS $$
BEGIN
    RETURN encode(
        sha256(
            (task_id::text || COALESCE(account_id::text, '') || COALESCE(wallet_id::text, '') || date_suffix::text)::bytea
        ),
        'hex'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to safely clean up old audit logs (retention policy)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_logs 
    WHERE created_at < CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Grant necessary permissions
GRANT ALL ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO postgres;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO postgres;

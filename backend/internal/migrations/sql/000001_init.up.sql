-- 000001_init.up.sql
-- Initial database schema for Web3AirdropOS

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    email_verified_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    ip_address VARCHAR(50),
    user_agent VARCHAR(500),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token_hash);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Refresh tokens table (for token rotation)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    family_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(id),
    ip_address VARCHAR(50),
    user_agent VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    chain VARCHAR(50) NOT NULL DEFAULT 'evm',
    wallet_type VARCHAR(50) NOT NULL DEFAULT 'generated',
    encrypted_private_key TEXT,
    group_id UUID,
    is_locked BOOLEAN DEFAULT false,
    locked_by UUID,
    locked_at TIMESTAMPTZ,
    balance VARCHAR(100),
    balance_usd DECIMAL(20, 8),
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wallets_user ON wallets(user_id);
CREATE INDEX idx_wallets_address ON wallets(address);
CREATE INDEX idx_wallets_group ON wallets(group_id);

-- Wallet groups table
CREATE TABLE IF NOT EXISTS wallet_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Platform accounts table
CREATE TABLE IF NOT EXISTS platform_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    username VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    platform_id VARCHAR(255),
    encrypted_credentials TEXT,
    linked_wallet_id UUID REFERENCES wallets(id),
    is_active BOOLEAN DEFAULT true,
    is_locked BOOLEAN DEFAULT false,
    locked_by UUID,
    locked_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_platform_accounts_user ON platform_accounts(user_id);
CREATE INDEX idx_platform_accounts_platform ON platform_accounts(platform);
CREATE UNIQUE INDEX idx_platform_accounts_unique ON platform_accounts(user_id, platform, username);

-- Proxies table
CREATE TABLE IF NOT EXISTS proxies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255),
    protocol VARCHAR(20) NOT NULL DEFAULT 'http',
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    username VARCHAR(255),
    encrypted_password TEXT,
    is_active BOOLEAN DEFAULT true,
    last_tested_at TIMESTAMPTZ,
    latency_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_proxies_user ON proxies(user_id);

-- Campaigns table
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL,
    platform_url TEXT,
    status VARCHAR(50) DEFAULT 'draft',
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_campaigns_user ON campaigns(user_id);
CREATE INDEX idx_campaigns_status ON campaigns(status);

-- Campaign tasks table
CREATE TABLE IF NOT EXISTS campaign_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    task_type VARCHAR(50) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    target_url TEXT,
    parameters JSONB,
    order_index INTEGER DEFAULT 0,
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_campaign_tasks_campaign ON campaign_tasks(campaign_id);

-- Task executions table
CREATE TYPE task_status AS ENUM ('pending', 'running', 'done', 'failed', 'manual_required', 'skipped');

CREATE TABLE IF NOT EXISTS task_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES campaign_tasks(id) ON DELETE CASCADE,
    wallet_id UUID REFERENCES wallets(id),
    account_id UUID REFERENCES platform_accounts(id),
    status task_status DEFAULT 'pending',
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    proof_type VARCHAR(50),
    proof_data JSONB,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_task_executions_task ON task_executions(task_id);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_idempotency ON task_executions(idempotency_key);

-- Secrets vault table
CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_type VARCHAR(50) NOT NULL,
    encrypted_value TEXT NOT NULL,
    iv VARCHAR(32) NOT NULL,
    metadata JSONB,
    version INTEGER DEFAULT 1,
    last_accessed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_secrets_user ON secrets(user_id);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    result VARCHAR(20) NOT NULL,
    ip_address VARCHAR(50),
    user_agent VARCHAR(500),
    request_data JSONB,
    response_data JSONB,
    error_message TEXT,
    duration_ms INTEGER,
    idempotency_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- Automation jobs table
CREATE TABLE IF NOT EXISTS automation_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL,
    schedule VARCHAR(100),
    is_active BOOLEAN DEFAULT false,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_automation_jobs_user ON automation_jobs(user_id);
CREATE INDEX idx_automation_jobs_next_run ON automation_jobs(next_run_at) WHERE is_active = true;

-- Job logs table
CREATE TABLE IF NOT EXISTS job_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    output JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_logs_job ON job_logs(job_id);

-- Content drafts table
CREATE TABLE IF NOT EXISTS content_drafts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'draft',
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(id),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_content_drafts_user ON content_drafts(user_id);
CREATE INDEX idx_content_drafts_status ON content_drafts(status);

-- Browser profiles table
CREATE TABLE IF NOT EXISTS browser_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    user_agent VARCHAR(500),
    proxy_id UUID REFERENCES proxies(id),
    cookies JSONB,
    local_storage JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_browser_profiles_user ON browser_profiles(user_id);

-- Browser sessions table
CREATE TABLE IF NOT EXISTS browser_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES browser_profiles(id) ON DELETE CASCADE,
    task_id UUID REFERENCES campaign_tasks(id),
    status VARCHAR(50) DEFAULT 'created',
    vnc_url VARCHAR(500),
    ws_url VARCHAR(500),
    screenshot_url VARCHAR(500),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_browser_sessions_profile ON browser_sessions(profile_id);
CREATE INDEX idx_browser_sessions_status ON browser_sessions(status);

-- Rate limit logs (for analytics)
CREATE TABLE IF NOT EXISTS rate_limit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(255) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    blocked BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_logs_identifier ON rate_limit_logs(identifier);
CREATE INDEX idx_rate_limit_logs_created ON rate_limit_logs(created_at);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_wallets_updated_at BEFORE UPDATE ON wallets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_wallet_groups_updated_at BEFORE UPDATE ON wallet_groups FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_platform_accounts_updated_at BEFORE UPDATE ON platform_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_proxies_updated_at BEFORE UPDATE ON proxies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_campaign_tasks_updated_at BEFORE UPDATE ON campaign_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_task_executions_updated_at BEFORE UPDATE ON task_executions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_secrets_updated_at BEFORE UPDATE ON secrets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_automation_jobs_updated_at BEFORE UPDATE ON automation_jobs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_content_drafts_updated_at BEFORE UPDATE ON content_drafts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_browser_profiles_updated_at BEFORE UPDATE ON browser_profiles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

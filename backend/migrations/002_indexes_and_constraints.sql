-- Migration: 002_indexes_and_constraints
-- Description: Additional indexes, constraints, and performance optimizations
-- Created: 2026-01-08

-- ============================================================================
-- COMPOSITE INDEXES FOR COMMON QUERIES
-- ============================================================================

-- Task execution queries (finding pending tasks for a campaign)
CREATE INDEX IF NOT EXISTS idx_task_executions_task_status ON task_executions(task_id, status);

-- User's recent audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);

-- Campaign progress queries
CREATE INDEX IF NOT EXISTS idx_task_executions_campaign_status ON task_executions(task_id, status)
INCLUDE (completed_at, proof_value);

-- Wallet transaction history
CREATE INDEX IF NOT EXISTS idx_transactions_wallet_created ON transactions(wallet_id, created_at DESC);

-- Scheduled posts to execute
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_pending ON scheduled_posts(scheduled_for, status)
WHERE status = 'pending';

-- Active automation jobs to run
CREATE INDEX IF NOT EXISTS idx_automation_jobs_active_next ON automation_jobs(next_run_at)
WHERE is_active = true AND deleted_at IS NULL;

-- Active browser sessions
CREATE INDEX IF NOT EXISTS idx_browser_sessions_active ON browser_sessions(user_id, status)
WHERE status IN ('starting', 'active', 'idle');

-- ============================================================================
-- ADDITIONAL CONSTRAINTS
-- ============================================================================

-- Ensure positive retry counts
ALTER TABLE task_executions ADD CONSTRAINT check_retry_count 
CHECK (retry_count >= 0 AND max_retries >= 0);

-- Ensure valid status transitions (via trigger)
CREATE OR REPLACE FUNCTION validate_task_status_transition()
RETURNS TRIGGER AS $$
BEGIN
    -- Allow any transition from NULL or PENDING
    IF OLD.status IS NULL OR OLD.status = 'PENDING' THEN
        RETURN NEW;
    END IF;
    
    -- RUNNING can go to MANUAL_REQUIRED, DONE, FAILED, or stay RUNNING
    IF OLD.status = 'RUNNING' THEN
        IF NEW.status NOT IN ('RUNNING', 'MANUAL_REQUIRED', 'DONE', 'FAILED') THEN
            RAISE EXCEPTION 'Invalid status transition from RUNNING to %', NEW.status;
        END IF;
        RETURN NEW;
    END IF;
    
    -- MANUAL_REQUIRED can go to DONE, FAILED, or RUNNING (resumed)
    IF OLD.status = 'MANUAL_REQUIRED' THEN
        IF NEW.status NOT IN ('MANUAL_REQUIRED', 'RUNNING', 'DONE', 'FAILED') THEN
            RAISE EXCEPTION 'Invalid status transition from MANUAL_REQUIRED to %', NEW.status;
        END IF;
        RETURN NEW;
    END IF;
    
    -- DONE and FAILED are terminal states (allow reset to PENDING for retry)
    IF OLD.status IN ('DONE', 'FAILED') THEN
        IF NEW.status NOT IN ('DONE', 'FAILED', 'PENDING') THEN
            RAISE EXCEPTION 'Invalid status transition from % to %', OLD.status, NEW.status;
        END IF;
        RETURN NEW;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_task_execution_status
BEFORE UPDATE OF status ON task_executions
FOR EACH ROW
EXECUTE FUNCTION validate_task_status_transition();

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- Campaign progress view
CREATE OR REPLACE VIEW campaign_progress_view AS
SELECT 
    c.id AS campaign_id,
    c.user_id,
    c.name AS campaign_name,
    c.status AS campaign_status,
    COUNT(DISTINCT ct.id) AS total_tasks,
    COUNT(DISTINCT CASE WHEN te.status = 'DONE' THEN te.id END) AS completed_executions,
    COUNT(DISTINCT CASE WHEN te.status = 'FAILED' THEN te.id END) AS failed_executions,
    COUNT(DISTINCT CASE WHEN te.status = 'RUNNING' THEN te.id END) AS running_executions,
    COUNT(DISTINCT CASE WHEN te.status = 'MANUAL_REQUIRED' THEN te.id END) AS manual_required,
    ROUND(
        CASE 
            WHEN COUNT(DISTINCT ct.id) > 0 
            THEN (COUNT(DISTINCT CASE WHEN te.status = 'DONE' THEN ct.id END)::DECIMAL / COUNT(DISTINCT ct.id)) * 100 
            ELSE 0 
        END, 2
    ) AS progress_percent
FROM campaigns c
LEFT JOIN campaign_tasks ct ON ct.campaign_id = c.id
LEFT JOIN task_executions te ON te.task_id = ct.id
WHERE c.deleted_at IS NULL
GROUP BY c.id, c.user_id, c.name, c.status;

-- User activity summary view
CREATE OR REPLACE VIEW user_activity_summary AS
SELECT 
    u.id AS user_id,
    u.email,
    COUNT(DISTINCT w.id) AS wallet_count,
    COUNT(DISTINCT pa.id) AS account_count,
    COUNT(DISTINCT c.id) AS campaign_count,
    COUNT(DISTINCT al.id) AS total_actions,
    MAX(al.created_at) AS last_action_at
FROM users u
LEFT JOIN wallets w ON w.user_id = u.id AND w.deleted_at IS NULL
LEFT JOIN platform_accounts pa ON pa.user_id = u.id AND pa.deleted_at IS NULL
LEFT JOIN campaigns c ON c.user_id = u.id AND c.deleted_at IS NULL
LEFT JOIN audit_logs al ON al.user_id = u.id
WHERE u.deleted_at IS NULL
GROUP BY u.id, u.email;

-- ============================================================================
-- RATE LIMITING SUPPORT (for IP-based rate limiting in addition to Redis)
-- ============================================================================

CREATE TABLE IF NOT EXISTS rate_limit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(200) NOT NULL, -- IP, user_id, or account_id
    endpoint VARCHAR(200) NOT NULL,
    request_count INTEGER DEFAULT 1,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rate_limit_logs_identifier ON rate_limit_logs(identifier, endpoint, window_start);

-- Cleanup old rate limit logs (run via cron)
CREATE OR REPLACE FUNCTION cleanup_rate_limit_logs()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM rate_limit_logs WHERE window_end < CURRENT_TIMESTAMP - INTERVAL '1 hour';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Record this migration
INSERT INTO schema_migrations (version, name, checksum) 
VALUES ('002', 'indexes_and_constraints', 'auto-generated')
ON CONFLICT (version) DO NOTHING;

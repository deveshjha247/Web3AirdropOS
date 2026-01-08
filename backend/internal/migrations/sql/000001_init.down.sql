-- 000001_init.down.sql
-- Rollback initial schema

DROP TRIGGER IF EXISTS update_browser_profiles_updated_at ON browser_profiles;
DROP TRIGGER IF EXISTS update_content_drafts_updated_at ON content_drafts;
DROP TRIGGER IF EXISTS update_automation_jobs_updated_at ON automation_jobs;
DROP TRIGGER IF EXISTS update_secrets_updated_at ON secrets;
DROP TRIGGER IF EXISTS update_task_executions_updated_at ON task_executions;
DROP TRIGGER IF EXISTS update_campaign_tasks_updated_at ON campaign_tasks;
DROP TRIGGER IF EXISTS update_campaigns_updated_at ON campaigns;
DROP TRIGGER IF EXISTS update_proxies_updated_at ON proxies;
DROP TRIGGER IF EXISTS update_platform_accounts_updated_at ON platform_accounts;
DROP TRIGGER IF EXISTS update_wallet_groups_updated_at ON wallet_groups;
DROP TRIGGER IF EXISTS update_wallets_updated_at ON wallets;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS rate_limit_logs;
DROP TABLE IF EXISTS browser_sessions;
DROP TABLE IF EXISTS browser_profiles;
DROP TABLE IF EXISTS content_drafts;
DROP TABLE IF EXISTS job_logs;
DROP TABLE IF EXISTS automation_jobs;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS task_executions;
DROP TABLE IF EXISTS campaign_tasks;
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS proxies;
DROP TABLE IF EXISTS platform_accounts;
DROP TABLE IF EXISTS wallet_groups;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS task_status;

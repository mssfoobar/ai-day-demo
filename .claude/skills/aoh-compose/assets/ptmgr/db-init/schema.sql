CREATE SCHEMA IF NOT EXISTS "ptmgr";
SET search_path TO "ptmgr";

CREATE TABLE fcm_token(
    id UUID NOT NULL,
    token_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    last_active_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (token_id),
    UNIQUE (user_id, device_id)
);

CREATE OR REPLACE FUNCTION set_updated_at()
    RETURNS TRIGGER
    LANGUAGE plpgsql AS
'
    BEGIN
        NEW.updated_at = now();
        RETURN NEW;
    END
';

CREATE OR REPLACE TRIGGER set_fcm_token_updated_at
    BEFORE UPDATE ON fcm_token
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

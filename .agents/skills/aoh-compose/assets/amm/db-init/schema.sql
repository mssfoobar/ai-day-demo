CREATE TABLE IF NOT EXISTS attachment (
    id UUID PRIMARY KEY,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    file_size INT NOT NULL,
    storage_key TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    scan_status TEXT NOT NULL,
    index_status TEXT NOT NULL,
    module TEXT NOT NULL,
    entity_id UUID,
    entity_type TEXT,
    checksum TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE,
    tenant_id TEXT NOT NULL,
    occ_lock INT NOT NULL DEFAULT 0,
    preview_data BYTEA,
    preview_mime_type TEXT,
    preview_status TEXT NOT NULL DEFAULT 'NA'
);

COMMENT ON COLUMN attachment.scan_status IS 'NA: scanning disabled, PENDING: pending antivirus scan, SUCCESS: scanned with no discovery, REJECTED: detected as malicious';
COMMENT ON COLUMN attachment.index_status IS 'NA: indexing disabled, PENDING: indexing pending, SUCCESS: indexed';

CREATE TABLE IF NOT EXISTS secure_download_id (
    download_id UUID PRIMARY KEY,
    attachment_id UUID NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk__attachment_id FOREIGN KEY (attachment_id) REFERENCES attachment(id),
    CONSTRAINT unique__attachment_id_download_id UNIQUE (attachment_id, download_id)
);

CREATE TABLE IF NOT EXISTS secure_download_id_user (
   download_id UUID NOT NULL,
   user_id UUID NOT NULL,
   PRIMARY KEY (download_id, user_id),
   CONSTRAINT fk__download_id FOREIGN KEY (download_id) REFERENCES secure_download_id(download_id)
);

CREATE TABLE IF NOT EXISTS data_access (
    attachment_id UUID NOT NULL,
    resource_name TEXT NOT NULL,
    scope TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT,
    PRIMARY KEY (attachment_id, resource_name, scope),
    CONSTRAINT fk__attachment_id FOREIGN KEY (attachment_id) REFERENCES attachment(id)
);

COMMENT ON COLUMN data_access.scope IS 'view: read access, edit: write access';

CREATE TABLE IF NOT EXISTS attachment_tag (
    attachment_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (attachment_id, tag_id),
    CONSTRAINT fk__attachment_id FOREIGN KEY (attachment_id) REFERENCES attachment(id)
);


CREATE FUNCTION set_aoh_amm_current_timestamp_updated_at() RETURNS trigger
    LANGUAGE plpgsql
AS
$$
DECLARE
    _new record;
BEGIN
    _new := NEW;
    _new."updated_at" = NOW();
    RETURN _new;
END;
$$;

CREATE TRIGGER set_aoh_amm_attachment_current_timestamp_updated_at
    BEFORE UPDATE
    ON attachment
    FOR EACH ROW
    EXECUTE FUNCTION set_aoh_amm_current_timestamp_updated_at();

CREATE TRIGGER set_aoh_amm_data_access_current_timestamp_updated_at
    BEFORE UPDATE
    ON data_access
    FOR EACH ROW
    EXECUTE FUNCTION set_aoh_amm_current_timestamp_updated_at();

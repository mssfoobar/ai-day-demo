CREATE TABLE IF NOT EXISTS message (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  ref_link TEXT,
  icon_id TEXT,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by TEXT NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by TEXT NOT NULL,
  occ_lock INT NOT NULL DEFAULT 0,
  tenant_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS message_user_mapper (
  message_id uuid NOT NULL REFERENCES message ON DELETE CASCADE,
  user_id uuid NOT NULL,
  status SMALLINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL default CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id, user_id)
);
COMMENT ON COLUMN "message_user_mapper"."status" IS '0: unread, 1: read, 2: deleted';

CREATE TABLE IF NOT EXISTS message_history (
  id UUID PRIMARY KEY,
  message_id UUID NOT NULL REFERENCES message ON DELETE CASCADE,
  user_id UUID NOT NULL,
  status SMALLINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON COLUMN "message_history"."status" IS '0: unread, 1: read, 2: deleted';

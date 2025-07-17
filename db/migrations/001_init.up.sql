CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- only once per DB

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    avatar_url TEXT,
    is_active BOOLEAN DEFAULT true,
    provider TEXT DEFAULT 'petrel',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    service TEXT NOT NULL, -- e.g 'notion', 'confluence'
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE notion_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL,
    workspace_name TEXT,
    workspace_icon TEXT,
    bot_id TEXT,
    notion_user_id TEXT,
    notion_user_name TEXT,
    notion_user_avatar TEXT,
    notion_user_email TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);
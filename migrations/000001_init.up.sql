CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE platform_admin (
    id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),

    status VARCHAR(30) NOT NULL DEFAULT 'active',

    password TEXT NOT NULL, 

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
);

CREATE TABLE platform_plans (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    plan_name VARCHAR(100) NOT NULL UNIQUE,
    note TEXT, 

    price NUMERIC(14,2) NOT NULL DEFAULT 0,

    status VARCHAR(30) NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    phone VARCHAR(20),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,

    role VARCHAR(30) NOT NULL,

    email VARCHAR(255) NOT NULL,
    password TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_user_email_unique
        UNIQUE (tenant_id, email)
);

CREATE TABLE platform_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL,

    current_plan_id UUID NOT NULL,

    current_plan_name VARCHAR(100) NOT NULL,

    end_date TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_subscription_unique
        UNIQUE (tenant_id)
);

CREATE TABLE platform_plan_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    tenant_id UUID NOT NULL,

    transaction_id VARCHAR(255) NOT NULL UNIQUE,

    payment_method VARCHAR(50) NOT NULL,

    plan_id UUID NOT NULL,

    amount NUMERIC(14,2) NOT NULL,

    status VARCHAR(30) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_user_tenant_id
ON tenant_user(tenant_id);

CREATE INDEX idx_subscription_tenant_id
ON platform_subscriptions(tenant_id);

CREATE INDEX idx_plan_transactions_tenant_id
ON platform_plan_transactions(tenant_id);
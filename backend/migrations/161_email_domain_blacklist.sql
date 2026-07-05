-- Persist email domain blacklist enforcement that was first applied manually in production.
CREATE TABLE IF NOT EXISTS email_domain_blacklist (
    id BIGSERIAL PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,
    reason TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    include_subdomains BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_domain_blacklist_domain_normalized CHECK (
        domain = lower(btrim(domain))
        AND domain <> ''
        AND domain !~ '[@[:space:]/]'
    )
);

CREATE OR REPLACE FUNCTION touch_email_domain_blacklist_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION enforce_user_email_domain_blacklist()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    email_domain TEXT;
    matched_domain TEXT;
BEGIN
    email_domain := lower(btrim(split_part(NEW.email, '@', 2)));

    SELECT b.domain
      INTO matched_domain
      FROM email_domain_blacklist b
     WHERE b.enabled
       AND (
            email_domain = b.domain
            OR (b.include_subdomains AND email_domain LIKE '%.' || b.domain)
       )
     ORDER BY length(b.domain) DESC
     LIMIT 1;

    IF matched_domain IS NOT NULL THEN
        RAISE EXCEPTION 'EMAIL_DOMAIN_BLACKLISTED: %', email_domain
            USING ERRCODE = '23514',
                  DETAIL = format('Matched blacklisted domain: %s', matched_domain),
                  HINT = 'Use a different email domain or disable the blacklist entry.';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_touch_email_domain_blacklist_updated_at ON email_domain_blacklist;
CREATE TRIGGER trg_touch_email_domain_blacklist_updated_at
    BEFORE UPDATE ON email_domain_blacklist
    FOR EACH ROW
    EXECUTE FUNCTION touch_email_domain_blacklist_updated_at();

DROP TRIGGER IF EXISTS trg_enforce_user_email_domain_blacklist ON users;
CREATE TRIGGER trg_enforce_user_email_domain_blacklist
    BEFORE INSERT OR UPDATE OF email ON users
    FOR EACH ROW
    EXECUTE FUNCTION enforce_user_email_domain_blacklist();

INSERT INTO email_domain_blacklist (domain, reason, enabled, include_subdomains)
VALUES (
    'web-library.net',
    'temporary/disposable email domain used for automated signups',
    TRUE,
    TRUE
)
ON CONFLICT (domain) DO NOTHING;

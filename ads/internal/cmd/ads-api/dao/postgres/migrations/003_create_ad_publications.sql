-- Every ad published through POST /ads, used to enforce the publication quota.
-- It is a log, not a reference to ads: deleting an ad must not give the quota back, so there is no foreign key.
CREATE TABLE ad_publications (
    id           BIGSERIAL   PRIMARY KEY,
    ad_id        BIGINT      NOT NULL,
    user_id      UUID        NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ad_publications_user_id_published_at_idx ON ad_publications (user_id, published_at);

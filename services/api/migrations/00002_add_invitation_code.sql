-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE invitations ADD COLUMN IF NOT EXISTS code VARCHAR(32) UNIQUE;
CREATE INDEX IF NOT EXISTS idx_invitations_code ON invitations(code);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP INDEX IF EXISTS idx_invitations_code;
ALTER TABLE invitations DROP COLUMN IF EXISTS code;

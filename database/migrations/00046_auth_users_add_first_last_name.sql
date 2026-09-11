-- +goose Up
-- Adds optional First/Last Name to a User, alongside the Email/Role/
-- Status/PasswordHash identity internal/auth/model.go already defines.
-- NOT NULL DEFAULT '' rather than nullable columns: "empty string means
-- unset" is the same shape every other optional text field in this
-- schema already uses (e.g. the columns 00044 kept on customers,
-- devices, etc.) -- a User can still be created, and keep operating,
-- with nothing but an email, exactly as today.
ALTER TABLE users ADD COLUMN first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN last_name TEXT NOT NULL DEFAULT '';

-- notes.author_first_name/author_last_name are a snapshot captured at
-- write time from the author's User record as it stood at that moment --
-- the exact same "not resolved via a live join against auth.User at read
-- time" reasoning notes.author_email already documents (see
-- internal/note.Note's own doc comment), extended to cover the two new
-- optional name fields a User can now set.
ALTER TABLE notes ADD COLUMN author_first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE notes ADD COLUMN author_last_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE notes DROP COLUMN author_last_name;
ALTER TABLE notes DROP COLUMN author_first_name;
ALTER TABLE users DROP COLUMN last_name;
ALTER TABLE users DROP COLUMN first_name;

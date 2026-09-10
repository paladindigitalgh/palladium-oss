-- +goose Up
-- Removes the free-text description column from every domain where it
-- was never actually surfaced anywhere but a create (or create/edit)
-- dialog -- an operator could type something into it, but nothing ever
-- showed it back to them again, so it was pure write-only data no one
-- could ever read. Scoped narrowly, at the user's own direction, to the
-- domains where that was true; every other domain's description (e.g.
-- customers, devices, OLTs, services, sites/buildings/rooms/racks,
-- access networks/interfaces, PON ports, and the Device/OLT Model
-- catalogs) keeps its column, since those are real, currently-displayed
-- fields.
ALTER TABLE customer_devices DROP COLUMN description;
ALTER TABLE service_equipment DROP COLUMN description;
ALTER TABLE products DROP COLUMN description;
ALTER TABLE providers DROP COLUMN description;
ALTER TABLE contacts DROP COLUMN description;
ALTER TABLE locations DROP COLUMN description;
ALTER TABLE connection_profiles DROP COLUMN description;
ALTER TABLE catalogs DROP COLUMN description;
ALTER TABLE service_profiles DROP COLUMN description;
ALTER TABLE provisioning_profiles DROP COLUMN description;

-- +goose Down
ALTER TABLE customer_devices ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE service_equipment ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE contacts ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE connection_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE catalogs ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE service_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE provisioning_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
-- Down restores the column, not any data that was in it -- dropping a
-- column is a real data loss, the same as every other DROP COLUMN in
-- this codebase's migration history.

-- +goose Up
-- UNIPort records which physical LAN port on the ONU/ONT a Service is
-- delivered over -- the operator-facing choice is "10GE" (uni 1) or
-- "1GE" (uni 2), never both (see internal/serviceequipment/model.go's
-- own doc comment on why this one narrow piece of delivery-configuration
-- detail now lives here, an intentional, documented exception to this
-- package's original "never how it is configured" scope). Real Kontron
-- provisioning needs it: the OLT command is "service-profile <profile>
-- uni <N>", and applying the wrong uni value either fails outright or
-- configures the wrong physical port.
--
-- Plain INTEGER, not an enum type or CHECK constraint -- the same
-- "validity is application-level" choice this table's own role column
-- already makes (see database/migrations/00014_serviceequipment_service_
-- equipment.sql): ServiceEquipment.Validate requires 1 or 2 for ONU/ONT
-- equipment, nothing else.
--
-- No existing row has a real answer for which port it uses, so this
-- backfills every one to 1 (10GE) -- a deliberate, arbitrary default, not
-- a claim about physical reality; an operator revisits it on the next
-- Edit Service Equipment / Add Service if that guess is wrong.
ALTER TABLE service_equipment ADD COLUMN uni_port INTEGER;
UPDATE service_equipment SET uni_port = 1;
ALTER TABLE service_equipment ALTER COLUMN uni_port SET NOT NULL;

-- +goose Down
ALTER TABLE service_equipment DROP COLUMN uni_port;

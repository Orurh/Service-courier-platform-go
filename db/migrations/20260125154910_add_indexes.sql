-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS ux_delivery_order_id
    ON delivery (order_id);

CREATE INDEX IF NOT EXISTS ix_delivery_courier_id
    ON delivery (courier_id);

CREATE INDEX IF NOT EXISTS ix_delivery_deadline_courier_id
    ON delivery (deadline, courier_id);

CREATE INDEX IF NOT EXISTS ix_couriers_status_id
    ON couriers (status, id);

-- +goose Down
DROP INDEX IF EXISTS ix_couriers_status_id;
DROP INDEX IF EXISTS ix_delivery_deadline_courier_id;
DROP INDEX IF EXISTS ix_delivery_courier_id;
DROP INDEX IF EXISTS ux_delivery_order_id;
-- +goose Up
ALTER TABLE couriers
    ADD COLUMN transport_type TEXT NOT NULL DEFAULT 'on_foot';

CREATE TABLE delivery (
    id          BIGSERIAL PRIMARY KEY,
    courier_id  BIGINT NOT NULL,
    order_id    VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT now(),
    deadline    TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS delivery;
ALTER TABLE couriers DROP COLUMN transport_type;

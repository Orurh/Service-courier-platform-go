-- +goose Up
CREATE TABLE IF NOT EXISTS delivery (
    id          BIGSERIAL PRIMARY KEY,
    courier_id  BIGINT NOT NULL REFERENCES couriers(id),
    order_id    VARCHAR(255) NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT now(),
    deadline    TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS delivery;

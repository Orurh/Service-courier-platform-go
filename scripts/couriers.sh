#!/usr/bin/env bash
set -euo pipefail

# Засеем БД тестовыми данными и проверим планы выполнения ключевых запросов.
COURIERS="${1:-10000}"
DELIVERIES="${2:-100000}"

DB="${POSTGRES_DB:-test_db}"
USER="${POSTGRES_USER:-myuser}"

PASS="$(docker compose exec -T postgres sh -lc 'cat /run/secrets/postgres_password')"

psqlc() {
  docker compose exec -T -e PGPASSWORD="$PASS" postgres \
    psql -h localhost -U "$USER" -d "$DB" -v ON_ERROR_STOP=1 "$@"
}

require_index() {
  local idx="$1"
  psqlc -Atc "select 1
              from pg_indexes
              where schemaname='public'
                and indexname='${idx}'" | grep -q '^1$' \
    || { echo "ERROR: missing index ${idx}"; exit 1; }
}

echo "1. Проверка подключения и текущей статистики БД"

psqlc -c "select current_database() as db,
                 current_user as usr,
                 (select count(*) from couriers) as couriers,
                 (select count(*) from delivery) as deliveries;"

echo "2. Засеем курьеров (если пусто) + вставим доставки (пропустим существующие order_ids)"

psqlc -c "begin;

          insert into couriers(name, phone, status, transport_type)
          select
            'Seed Courier #' || gs,
            '+7' || lpad(gs::text, 10, '0'),
            case when gs % 3 = 0 then 'available'
                 when gs % 3 = 1 then 'busy'
                 else 'paused' end,
            case when gs % 4 = 0 then 'on_foot'
                 when gs % 4 = 1 then 'bike'
                 when gs % 4 = 2 then 'car'
                 else 'scooter' end
          from generate_series(1, ${COURIERS}) gs
          where not exists (select 1 from couriers);

          do \$\$
          declare
            max_courier_id bigint;
          begin
            select max(id) into max_courier_id from couriers;
            if max_courier_id is null then
              raise exception 'couriers is empty';
            end if;

            insert into delivery(courier_id, order_id, assigned_at, deadline)
            select
              (1 + floor(random() * max_courier_id))::bigint as courier_id,
              'ORD-' || lpad(gs::text, 10, '0') as order_id,
              now() - (random() * interval '7 days') as assigned_at,
              case when (gs % 2) = 0
                   then now() - (random() * interval '7 days')  -- часть просрочена
                   else now() + (random() * interval '7 days')  -- часть в будущем
              end as deadline
            from generate_series(1, ${DELIVERIES}) gs
            on conflict (order_id) do nothing;
          end
          \$\$;

          commit;"

echo "3. Анализ таблиц для актуализации статистики"

psqlc -c "analyze couriers; analyze delivery;"

echo "4. Проверка индексов (наличие + список по таблицам)"
# Список индексов по конкретной таблице
psqlc -c "select indexname, indexdef
          from pg_indexes
          where schemaname='public' and tablename='couriers'
          order by indexname;"

# Явная проверка наличия нужных индексов 
require_index "ux_delivery_order_id"
require_index "ix_delivery_courier_id"
require_index "ix_delivery_deadline_courier_id"
require_index "ix_couriers_status_id"


psqlc -c "\di+ couriers*"
psqlc -c "\di+ delivery*"

echo "5. Проверка планов выполнения ключевых запросов (EXPLAIN ANALYZE)"
echo "  - GetByOrderID / DeleteByOrderID path (delivery.order_id)"
OID="$(psqlc -Atc "select order_id from delivery limit 1;")"
psqlc -c "explain (analyze, buffers)
          select id, courier_id, order_id, assigned_at, deadline
          from delivery
          where order_id = '${OID}';"

echo "  - ReleaseCouriers path (delivery.deadline[, courier_id])"
psqlc -c "explain (analyze, buffers)
          select d.courier_id
          from delivery d
          where d.deadline < now();"

echo "  - FindAvailableCourierForUpdate COUNT(*) subquery path (delivery.courier_id)"
CID="$(psqlc -Atc "select courier_id from delivery limit 1;")"
psqlc -c "explain (analyze, buffers)
          select count(*)
          from delivery d
          where d.courier_id = ${CID};"

echo "DONE"

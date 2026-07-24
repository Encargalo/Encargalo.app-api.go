# Grafana — queries OLAP (v1)

Datasource: Postgres OLAP (schema `analytics`).

Variables sugeridas: `$from`, `$to` (Grafana time range).

## GMV / ventas del periodo (órdenes Completed)

```sql
SELECT COALESCE(SUM(gmv), 0) AS gmv
FROM analytics.cube_order
WHERE status = 'Completed'
  AND created_at >= $__timeFrom()
  AND created_at < $__timeTo();
```

## Órdenes creadas / completadas / rechazadas / canceladas

```sql
SELECT
  COUNT(*) FILTER (WHERE TRUE) AS created,
  COUNT(*) FILTER (WHERE status = 'Completed') AS completed,
  COUNT(*) FILTER (WHERE status = 'Rejected') AS rejected,
  COUNT(*) FILTER (WHERE status = 'Cancelled') AS cancelled
FROM analytics.cube_order
WHERE created_at >= $__timeFrom()
  AND created_at < $__timeTo();
```

## Serie diaria GMV + órdenes

```sql
SELECT
  order_date AS time,
  SUM(gmv) FILTER (WHERE status = 'Completed') AS gmv,
  COUNT(*) AS orders_created,
  COUNT(*) FILTER (WHERE status = 'Completed') AS orders_completed
FROM analytics.cube_order
WHERE created_at >= $__timeFrom()
  AND created_at < $__timeTo()
GROUP BY order_date
ORDER BY order_date;
```

## Top 10 negocios por GMV

```sql
SELECT s.name, SUM(o.gmv) AS gmv, COUNT(*) AS orders
FROM analytics.cube_order o
JOIN analytics.dim_shop s ON s.shop_id = o.shop_id
WHERE o.status = 'Completed'
  AND o.created_at >= $__timeFrom()
  AND o.created_at < $__timeTo()
GROUP BY s.name
ORDER BY gmv DESC
LIMIT 10;
```

## Top 10 clientes por gasto

```sql
SELECT u.name, u.phone, SUM(o.gmv) AS gmv, COUNT(*) AS orders
FROM analytics.cube_order o
JOIN analytics.dim_user u ON u.user_id = o.customer_id
WHERE o.status = 'Completed'
  AND o.created_at >= $__timeFrom()
  AND o.created_at < $__timeTo()
GROUP BY u.name, u.phone
ORDER BY gmv DESC
LIMIT 10;
```

## Top 10 riders por órdenes Completed

```sql
SELECT r.name, r.phone, COUNT(*) AS completed_orders
FROM analytics.cube_order o
JOIN analytics.dim_rider r ON r.rider_id = o.rider_id
WHERE o.status = 'Completed'
  AND o.rider_id IS NOT NULL
  AND o.created_at >= $__timeFrom()
  AND o.created_at < $__timeTo()
GROUP BY r.name, r.phone
ORDER BY completed_orders DESC
LIMIT 10;
```

## Altas de usuarios en el rango

```sql
SELECT COUNT(*) AS registered_users
FROM analytics.dim_user
WHERE registered_at >= $__timeFrom()
  AND registered_at < $__timeTo();
```

## Altas de riders en el rango

```sql
SELECT COUNT(*) AS registered_riders
FROM analytics.dim_rider
WHERE registered_at >= $__timeFrom()
  AND registered_at < $__timeTo();
```

## Tasa de rechazo

```sql
SELECT
  COUNT(*) FILTER (WHERE status = 'Rejected')::float
    / NULLIF(COUNT(*), 0) AS rejection_rate
FROM analytics.cube_order
WHERE created_at >= $__timeFrom()
  AND created_at < $__timeTo();
```

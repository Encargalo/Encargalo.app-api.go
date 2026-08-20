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

## Tracking — pixels / CTAs (`fact_tracking_event`)

Los campos de negocio viven en `params` JSONB. Convención v1 del cliente (el backend no la impone):

- `type = 'pixel'` y `params.name = 'app_open'` — abrió la app
- `type = 'pixel'` y `params.name = 'shop_view'` + `params.shop_id` — entró a un local
- `type = 'cta'` — botones / llamadas a la acción

`user_id` y `session_id` los inyecta la API en `params`. Día en `America/Caracas`.

### Usuarios únicos que abrieron la app

```sql
SELECT COUNT(DISTINCT params->>'user_id') AS users
FROM analytics.fact_tracking_event
WHERE type = 'pixel'
  AND params->>'name' = 'app_open'
  AND created_at >= $__timeFrom()
  AND created_at < $__timeTo();
```

### Serie diaria: abrió la app

```sql
SELECT
  (created_at AT TIME ZONE 'America/Caracas')::date AS time,
  COUNT(DISTINCT params->>'user_id') AS users
FROM analytics.fact_tracking_event
WHERE type = 'pixel'
  AND params->>'name' = 'app_open'
  AND created_at >= $__timeFrom()
  AND created_at < $__timeTo()
GROUP BY 1
ORDER BY 1;
```

### Quiénes abrieron la app

```sql
SELECT DISTINCT u.user_id, u.name, u.phone
FROM analytics.fact_tracking_event e
JOIN analytics.dim_user u ON u.user_id = (e.params->>'user_id')::uuid
WHERE e.type = 'pixel'
  AND e.params->>'name' = 'app_open'
  AND e.created_at >= $__timeFrom()
  AND e.created_at < $__timeTo()
ORDER BY u.name;
```

### Usuarios únicos que visitaron un local (`$shop_id`)

```sql
SELECT COUNT(DISTINCT params->>'user_id') AS users
FROM analytics.fact_tracking_event
WHERE type = 'pixel'
  AND params->>'name' = 'shop_view'
  AND params->>'shop_id' = '$shop_id'
  AND created_at >= $__timeFrom()
  AND created_at < $__timeTo();
```

### Quiénes visitaron un local

```sql
SELECT DISTINCT u.user_id, u.name, u.phone
FROM analytics.fact_tracking_event e
JOIN analytics.dim_user u ON u.user_id = (e.params->>'user_id')::uuid
WHERE e.type = 'pixel'
  AND e.params->>'name' = 'shop_view'
  AND e.params->>'shop_id' = '$shop_id'
  AND e.created_at >= $__timeFrom()
  AND e.created_at < $__timeTo()
ORDER BY u.name;
```

### Visitó el local y compró vs no compró

```sql
SELECT
  COUNT(DISTINCT e.params->>'user_id') AS visitors,
  COUNT(DISTINCT o.customer_id) AS bought,
  COUNT(DISTINCT e.params->>'user_id')
    FILTER (WHERE o.order_id IS NULL) AS visited_no_buy
FROM analytics.fact_tracking_event e
LEFT JOIN analytics.cube_order o
  ON o.customer_id = (e.params->>'user_id')::uuid
 AND o.shop_id = (e.params->>'shop_id')::uuid
 AND o.created_at >= $__timeFrom()
 AND o.created_at < $__timeTo()
WHERE e.type = 'pixel'
  AND e.params->>'name' = 'shop_view'
  AND e.params->>'shop_id' = '$shop_id'
  AND e.created_at >= $__timeFrom()
  AND e.created_at < $__timeTo();
```

### Quiénes visitaron y no compraron

```sql
SELECT DISTINCT u.user_id, u.name, u.phone
FROM analytics.fact_tracking_event e
JOIN analytics.dim_user u ON u.user_id = (e.params->>'user_id')::uuid
LEFT JOIN analytics.cube_order o
  ON o.customer_id = u.user_id
 AND o.shop_id = (e.params->>'shop_id')::uuid
 AND o.created_at >= $__timeFrom()
 AND o.created_at < $__timeTo()
WHERE e.type = 'pixel'
  AND e.params->>'name' = 'shop_view'
  AND e.params->>'shop_id' = '$shop_id'
  AND e.created_at >= $__timeFrom()
  AND e.created_at < $__timeTo()
  AND o.order_id IS NULL
ORDER BY u.name;
```

### Panel genérico (cualquier `type` / `params.name`)

Variables Grafana: `$type`, `$event_name`.

```sql
SELECT
  (created_at AT TIME ZONE 'America/Caracas')::date AS time,
  COUNT(*) AS events,
  COUNT(DISTINCT params->>'user_id') AS users
FROM analytics.fact_tracking_event
WHERE type = '$type'
  AND params->>'name' = '$event_name'
  AND created_at >= $__timeFrom()
  AND created_at < $__timeTo()
GROUP BY 1
ORDER BY 1;
```

## ADDED Requirements

### Requirement: Historial de transacciones paginado
El sistema SHALL exponer `GET /riders/transactions` (autenticación requerida, rol `Rider`) que retorne el historial de movimientos financieros del rider autenticado con paginación por cursor o por página.

Query params:
- `page` (int, opcional, default 1): número de página
- `limit` (int, opcional, default 20, máx 50): registros por página

Respuesta:
```json
{
  "transactions": [
    {
      "id": "uuid",
      "order_id": "uuid",
      "amount": -8.00,
      "movement_type": "ride_cash",
      "payment_method": "cash",
      "distance_km": 3.2,
      "created_at": "2026-06-14T20:00:00Z"
    }
  ],
  "total": 45,
  "page": 1,
  "limit": 20
}
```

Las transacciones se retornan ordenadas por `created_at DESC`.

#### Scenario: Rider consulta su historial
- **WHEN** un rider autenticado hace `GET /riders/transactions`
- **THEN** el sistema retorna HTTP 200 con la primera página de transacciones ordenadas por fecha descendente

#### Scenario: Paginación funciona correctamente
- **WHEN** un rider autenticado hace `GET /riders/transactions?page=2&limit=10`
- **THEN** el sistema retorna HTTP 200 con los registros del 11 al 20 en orden descendente

#### Scenario: Rider sin transacciones
- **WHEN** un rider autenticado hace `GET /riders/transactions` y no tiene movimientos registrados
- **THEN** el sistema retorna HTTP 200 con `transactions = []` y `total = 0`

#### Scenario: Limit excede el máximo
- **WHEN** un rider hace `GET /riders/transactions?limit=100`
- **THEN** el sistema aplica el límite máximo de 50 registros por página

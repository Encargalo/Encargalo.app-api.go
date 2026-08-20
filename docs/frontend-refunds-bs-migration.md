# Front — Devoluciones y créditos en Bs

Delta de contrato para **app cliente** y **CRM staff**.  
No cambia URLs, estados ni el flujo `pending → accepted → completed | rejected`.  
Cambia **en qué moneda se leen y muestran** los montos de crédito / CTA / pago de devolución.

Guías completas (post-cambio):

- [`frontend-balance-refunds.md`](./frontend-balance-refunds.md)
- [`frontend-payment-review.md`](./frontend-payment-review.md)

Auth: cookie `encargalo-session`. Base URL local: `http://localhost:3000`.

---

## 1. Qué hay que hacer

1. **Renombrar campos** en tipos, parsers y UI (tabla de abajo).
2. **Mostrar Bs como moneda de la operación** (CTA, crédito al balance, monto a transferir en CRM).
3. **USD solo como equivalente informativo** (`amount_usd` / `balance_usd`), nunca como input ni como monto a pagar.
4. **Completar devoluciones contra `amount_bs`** (± 0.01), no contra USD.

Si el front todavía lee `*_usd` en estos campos, el CTA, el historial y el complete del CRM van a mostrar mal o a fallar.

---

## 2. Mapa de campos (breaking)

Buscar en el código y reemplazar. No dejar fallback al nombre viejo salvo `pago_movil_amount_usd` (solo pedidos ya persistidos; el API nuevo **no lo emite**).

| Superficie | Quitar | Usar |
|---|---|---|
| `GET /users/balance` | `balance`, `amount` | `balance_bs` + `balance_usd`, `amount_bs` + `amount_usd` |
| Historial / CTA | `refundable_pm_usd` | `refundable_pm_bs` |
| `order.details.payment` | `credited_usd` | `credited_bs` |
| `order.details.payment` | `refundable_pm_usd` | `refundable_pm_bs` |
| `order.details.payment` | `pago_movil_amount_usd` | `pago_movil_amount_bs` |
| `order.details.payment` | `balance_amount_usd` | `balance_amount_bs` |
| Checkout `POST /orders` | `balance_amount_usd` | `balance_amount_bs` |
| HTTP refund (`GET …/refund-requests`) | `amount_usd` | `amount_bs` (el HTTP **no** trae USD) |
| WS `refund_request_update` | usar `amount_usd` para pagar | `amount_bs` = monto a transferir; `amount_usd` informativo |
| CRM complete `POST …/complete` | `amount_usd` | `amount_bs` |
| WS staff `complete` | `amount_usd` | `amount_bs` |
| WS `payment_review_update` | `pago_movil_amount_usd` | `pago_movil_amount_bs` |

Campos **nuevos** (opcionales, `omitempty`):

| Campo | Dónde | Para qué |
|---|---|---|
| `details.payment.bcv_rate` | `order_update` | Tasa congelada del pedido (quote/review) |
| `details.payment.pago_movil_amount_bs` | `order_update` | Split PM en Bs al crear la orden |

---

## 3. App cliente

### 3.1 Balance

`GET /users/balance`

```json
{
  "balance_bs": 2080,
  "balance_usd": 52,
  "bcv_rate": 40,
  "transactions": [
    {
      "id": "…",
      "amount_bs": 2080,
      "amount_usd": 52,
      "movement_type": "order_rejected_refund",
      "refundable_pm_bs": 2080,
      "can_request_refund": true,
      "refund_eligible_until": "2026-07-18T19:28:36Z"
    }
  ]
}
```

| UI | Campo |
|---|---|
| Saldo grande | `balance_bs` (Bs). `balance_usd` opcional al lado |
| Línea del movimiento | `amount_bs` (signo según tipo). No usar `amount` |
| Monto del CTA “solicitar devolución” | **`refundable_pm_bs`** — no `amount_bs` ni `credited_bs` (pueden ser el total de la orden, no solo PM) |
| Mostrar CTA | solo si `can_request_refund === true` |
| Countdown | `refund_eligible_until` |

El umbral sigue siendo **más de $1**, pero el backend lo evalúa en Bs con la tasa **congelada** del crédito. El front no recalcula el mínimo: si no viene `can_request_refund`, no hay CTA.

### 3.2 Checkout

`POST /orders`: enviar `balance_amount_bs` (bolívares del saldo), no `balance_amount_usd`.

Mínimo a usar de balance: equivalente a **$1** (el API valida). Máximo: `min(balance_bs, total en Bs del quote)`.

### 3.3 CTA desde `order_update`

Misma regla que el historial. En `order.details.payment`:

```json
{
  "status": "refunded_shop_reject",
  "credited_bs": 2080,
  "balance_amount_bs": 400,
  "pago_movil_amount_bs": 1680,
  "bcv_rate": 40,
  "transaction_id": "…",
  "refundable_pm_bs": 1680,
  "can_request_refund": true,
  "refund_eligible_until": "…"
}
```

| Copy / dato | Campo |
|---|---|
| “Te acreditamos X a tu balance” | `credited_bs` (Bs) |
| Monto del CTA | `refundable_pm_bs` (Bs, solo PM) |
| `transaction_id` para `POST /users/balance/refund-requests` | `transaction_id` |

Estados de `details.payment.status` **no cambian**: `overpaid`, `rejected_underpaid`, `rejected_invalid`, `refunded_shop_reject`.

### 3.4 Solicitud HTTP

Sin cambio de payload:

```json
{ "transaction_id": "…" }
```

La respuesta de `GET /users/balance/refund-requests` y `GET …/:id` trae **`amount_bs`**. No hay `amount_usd` en HTTP.

### 3.5 WS `refund_request_update` (mismo `orders/manager`)

```json
{
  "type": "refund_request_update",
  "amount_bs": 1680,
  "amount_usd": 42,
  "bcv_rate": 40,
  "status": "pending"
}
```

- Mostrar / refrescar con `amount_bs`.
- `amount_usd` + `bcv_rate` son informativos (tasa **spot** al armar el broadcast). Pueden faltar si BCV no responde.
- Al `pending` / `accepted` / `completed` / `rejected` → refrescar `GET /users/balance`.

---

## 4. CRM staff

### 4.1 Cola de devoluciones

WS `users/staff/refund-requests`. Cada card:

| Dato en UI | Campo |
|---|---|
| Monto a pagar por Pago Móvil | **`amount_bs`** |
| Equivalente USD (opcional) | `amount_usd` |
| Validar complete | `amount_bs` ± 0.01 |

Ya no calcular `amount_bs = amount_usd × bcv_rate`. `amount_bs` viene listo; es lo que se transfiere.

### 4.2 Completar pago

`POST /users/staff/refund-requests/:id/complete`

```json
{
  "amount_bs": 1680,
  "proof_image": "data:image/jpeg;base64,…"
}
```

- Input del modal en **Bs**, verde/rojo contra `amount_bs` de la solicitud.
- Si el WS `complete` sigue mandando `amount_usd`, deja de funcionar (`422`).

### 4.3 Payment review (cola de comprobantes)

WS `payment_review_update`:

| Quitar | Usar |
|---|---|
| `pago_movil_amount_usd` | `pago_movil_amount_bs` |
| (si se mostraba crédito en USD) | `credited_bs` |

`expected_amount_bs` / `observed_amount_bs` / `difference_bs` no cambian.  
`total_usd` sigue siendo el total de la orden en USD (el catálogo no migró).

---

## 5. Copy

Dejar de hablar del crédito / devolución en dólares operativos.

| Antes | Ahora |
|---|---|
| “Te acreditamos $X a tu balance” | “Te acreditamos X Bs a tu balance” |
| “Solicitar devolución de $X” | “Solicitar devolución de X Bs” |
| Input complete CRM en USD | Input en Bs |
| “Pagamos $X por Pago Móvil” | “Pagamos X Bs por Pago Móvil” |

USD puede seguir en subtítulo (`≈ $Y`) usando `amount_usd` / `balance_usd` / `total_usd` donde existan.

---

## 6. Qué no cambia

- Rutas HTTP y sockets.
- Roles y auth.
- Ventana de 15 min, hold al crear, Discord cada 5 min.
- `can_request_refund` / `refund_eligible_until` / `transaction_id`.
- Flujo: registrar PM (`PUT /users/pago-movil`) **al crear**, no para mostrar el CTA.
- `order.total_price` / `delivery_fee` siguen en USD; `total_price_bs` / `delivery_fee_bs` ya existían.
- Tipos de movimiento (`payment_overage`, `payment_partial_credit`, `order_rejected_refund`, `balance_refund_*`).

---

## 7. Checklist

### App

- [ ] Tipos/DTOs: borrar `refundable_pm_usd`, `credited_usd`, `pago_movil_amount_usd`, `balance_amount_usd`, `balance` plano, `amount` plano en balance.
- [ ] Historial: CTA con `refundable_pm_bs`; saldo con `balance_bs`.
- [ ] `order_update`: CTA y copy de crédito en Bs (`credited_bs`, `refundable_pm_bs`).
- [ ] Checkout: `balance_amount_bs`.
- [ ] Sheet de solicitud: mostrar `amount_bs` (HTTP no trae USD).
- [ ] Copy de overpay / underpaid / shop reject en Bs.

### CRM

- [ ] Cola devoluciones: pagar y validar `amount_bs`.
- [ ] Complete HTTP/WS: body `amount_bs`.
- [ ] Payment review: `pago_movil_amount_bs` (no `*_usd`).
- [ ] Input de monto en Bs (verde/rojo ± 0.01).

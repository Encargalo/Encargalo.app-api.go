# Guía Frontend — Balance Bs y devoluciones en efectivo

Contrato para la **app cliente** y el **CRM staff**.  
No cubre payment review / quotes (eso está en [`frontend-payment-review.md`](./frontend-payment-review.md)).

Auth en todos los endpoints: cookie `encargalo-session`.  
Base URL local: `http://localhost:3000`.

---

## 1. Idea de producto

Cuando el cliente recibe un crédito en balance (rechazo del local, overpay, underpaid, etc.), la **porción que pagó por Pago Móvil** puede pedirse como **devolución en efectivo** durante **15 minutos**.

Flujo de la solicitud:

```
pending  →  accepted  →  completed
              ↓
           rejected  (restaura balance)
```

| Estado | Qué pasó | `movement_type` del débito | Balance |
|---|---|---|---|
| `pending` | Cliente pidió; saldo **retenido** | `balance_refund_hold` | −amount |
| `accepted` | Staff aceptó; falta pagar por Pago Móvil | `balance_refund_processing` | sin cambio |
| `completed` | Staff pagó y subió comprobante (`proof_url`) | `balance_refund_payout` | sin cambio |
| `rejected` | Staff rechazó; se **abona de nuevo** (`balance_refund_restore`) | hold queda histórico | +amount |

Reglas:

- El dinero se **reserva (debita) al crear** la solicitud, para que el cliente no pueda gastarlo en otra compra mientras staff responde.
- Tener Pago Móvil registrado **no** es requisito para mostrar el CTA; se exige al **crear** la solicitud.
- La ventana de 15 min **solo aplica para crear**. Una vez `pending`/`accepted`, no expira sola.
- Mínimo refundable: **más de $1** USD, evaluado en Bs con la tasa BCV **congelada** del crédito (no la tasa spot).

Hay **dos vías** para el CTA de “solicitar devolución”:

1. WS `order_update` → `details.payment.*`
2. Historial `GET /users/balance`

Y **una vía** para ver cambios de estado de la solicitud ya creada:

3. WS `refund_request_update` en el **mismo** socket `orders/manager`

---

## 2. Mapa de sockets (importante)

| Quién | URL | Para qué |
|---|---|---|
| **Cliente** | `ws(s)://…/orders/manager` | Órdenes (`order_update`) **y** estados de devolución (`refund_request_update`) |
| **CRM Staff** | `ws(s)://…/users/staff/refund-requests` | Cola de devoluciones + accept / reject |
| CRM Staff (HTTP) | `POST /users/staff/refund-requests/:id/complete` | Completar pago (imagen + monto) — **no** va por el WS de orders |

```
Cliente ──orders/manager──► order_update
                        └──► refund_request_update  (pending/accepted/completed/rejected)

CRM    ──staff/refund-requests──► snapshot + refund_request_update
     │                          └── accept / reject (WS)
     └── POST …/complete ──► sube imagen, valida monto, completed
                              └── también empuja refund_request_update al cliente
```

---

## 3. Cuándo se acredita balance

| Origen | `movement_type` | Cash-refundable |
|---|---|---|
| Staff aprobó con exceso | `payment_overage` | Todo el crédito |
| Staff rechazó por pago incompleto | `payment_partial_credit` | La porción PM observada |
| Local rechazó / timeout 10 min Pending | `order_rejected_refund` | **Solo** la porción Pago Móvil (`total_bs − balance_used_bs`). El crédito al balance es el total (incluye lo que ya era balance), pero `refundable_pm_bs` / CTA **nunca** incluyen lo pagado con balance. Si fue 100% balance → no elegible |

---

## 4. Cliente — Balance

### `GET /users/balance`

```json
{
  "balance_bs": 2080,
  "balance_usd": 52,
  "bcv_rate": 40,
  "transactions": [
    {
      "id": "b7b91d01-d7d6-4636-b30b-0b1716ca86ea",
      "amount_bs": 2080,
      "amount_usd": 52,
      "movement_type": "order_rejected_refund",
      "order_id": "9dcf4657-73ae-4214-9582-ba8259929d2e",
      "created_at": "2026-07-18T19:13:36Z",
      "refundable_pm_bs": 2080,
      "can_request_refund": true,
      "refund_eligible_until": "2026-07-18T19:28:36Z"
    }
  ]
}
```

| Campo | Uso en UI |
|---|---|
| `id` | Es el `transaction_id` para crear la solicitud |
| `can_request_refund` | Solo viene si es `true` (`omitempty`). **Si no está → no mostrar CTA** |
| `refund_eligible_until` | Countdown: `until - Date.now()` |
| `refundable_pm_bs` | **Monto del CTA / solicitud** (solo Pago Móvil). No uses `amount_bs` ni `credited_bs` del crédito — esos pueden ser el total de la orden |
| `refund_request_id` | En débitos `balance_refund_hold` / `processing` / `payout` y créditos `balance_refund_restore`: abrir detalle |

`can_request_refund` considera: ventana 15 min, monto > $1 equivalente en Bs a la tasa congelada, saldo actual del ledger, y que no haya ya una solicitud activa. **No** exige Pago Móvil.

Al crear la solicitud llega `refund_request_update` con `pending` → **refrescar balance** (ya bajó; tipo `hold`).  
Al recibir `accepted` / `completed` → **refrescar balance** (mismo débito cambia a `processing` / `payout`; el monto no cambia).  
Al recibir `rejected` → **refrescar balance** (se restauró).

---

## 5. Cliente — Pago Móvil

### `GET /users/pago-movil`
- `200` con datos registrados  
- `404` si nunca lo registró  

### `PUT /users/pago-movil`

```json
{
  "phone": "04141234567",
  "bank_code": "0102",
  "national_id": "V12345678"
}
```

Flujo UX al pedir devolución:

1. Usuario toca “Solicitar devolución”.
2. Si `GET /pago-movil` → 404: pedir registro → `PUT`.
3. Luego `POST /users/balance/refund-requests`.

---

## 6. Cliente — Solicitudes de devolución (HTTP)

### `POST /users/balance/refund-requests`

```json
{ "transaction_id": "b7b91d01-d7d6-4636-b30b-0b1716ca86ea" }
```

Tras crear, el cliente y el CRM reciben `refund_request_update` con `status: "pending"`. El balance **ya bajó** (movimiento `balance_refund_hold`). Al aceptar, ese mismo movimiento pasa a `balance_refund_processing`; al completar, a `balance_refund_payout` (solo entonces significa “ya pagado”).

Errores:

| Error / status | Significado |
|---|---|
| `422` + “debe registrar un pago móvil…” | Falta PM → mostrar formulario |
| `422` devolución no elegible | Ventana vencida / monto / tipo no válido |
| `409` ya existe solicitud | Ya pidió para ese crédito |
| `404` | Transacción ajena o inexistente |

### `GET /users/balance/refund-requests`  
Lista del cliente.

### `GET /users/balance/refund-requests/:id`  
Detalle (incluye `proof_url` cuando `status === "completed"`).

Respuesta típica:

```json
{
  "id": "…",
  "transaction_id": "…",
  "order_id": "…",
  "amount_usd": 52,
  "status": "completed",
  "pm_phone": "04141234567",
  "pm_bank_code": "0102",
  "pm_national_id": "V12345678",
  "proof_url": "https://res.cloudinary.com/…/image/upload/…",
  "rejection_reason": null,
  "created_at": "…",
  "expires_at": "…",
  "accepted_at": "…",
  "completed_at": "…",
  "resolved_at": "…"
}
```

---

## 7. Cliente — WebSocket `orders/manager`

Misma conexión de siempre. El handler debe discriminar por `type`.

### 7.1 CTA al acreditar — `order_update`

Cuando llega un crédito cash-refundable, `order.details.payment` puede traer:

```json
{
  "type": "order_update",
  "order": {
    "details": {
      "payment": {
        "status": "refunded_shop_reject",
        "credited_bs": 2080,
        "transaction_id": "b7b91d01-…",
        "refundable_pm_bs": 2080,
        "can_request_refund": true,
        "refund_eligible_until": "2026-07-18T19:28:36Z"
      }
    }
  }
}
```

| `details.payment.status` | Notas |
|---|---|
| `refunded_shop_reject` | Rechazo local / timeout |
| `overpaid` | Exceso acreditado |
| `rejected_underpaid` | `transaction_id` = crédito PM (`payment_partial_credit`) |

Si `can_request_refund === true` → mismo CTA que en historial.  
Si el usuario no estaba conectado → refrescar con `GET /users/balance`.

### 7.2 Estados de la solicitud — `refund_request_update`

Cuando se crea, acepta, rechaza o completa una solicitud, el **cliente dueño** recibe en el mismo socket:

```json
{
  "type": "refund_request_update",
  "id": "a1b2c3d4-…",
  "customer_id": "…",
  "transaction_id": "…",
  "order_id": "…",
  "amount_usd": 52,
  "amount_bs": 1872.00,
  "bcv_rate": 36.00,
  "status": "accepted",
  "pm_phone": "04141234567",
  "pm_bank_code": "0102",
  "pm_national_id": "V12345678",
  "proof_url": null,
  "rejection_reason": null,
  "created_at": "2026-07-18T19:13:36Z",
  "expires_at": "2026-07-18T19:28:36Z"
}
```

| `status` | UI sugerida | `movement_type` en historial |
|---|---|---|
| `pending` | “Retenido / en revisión”; **refrescar balance** | `balance_refund_hold` |
| `accepted` | “Aceptada — te estamos pagando” | `balance_refund_processing` |
| `completed` | “Pagada”; mostrar `proof_url` | `balance_refund_payout` |
| `rejected` | “Rechazada”; mostrar `rejection_reason`; **refrescar balance** | + `balance_refund_restore` |

**Pseudo-handler:**

```ts
ws.onmessage = (ev) => {
  const msg = JSON.parse(ev.data)
  if (msg.type === 'order_update') {
    // CTA si msg.order.details?.payment?.can_request_refund
  }
  if (msg.type === 'refund_request_update') {
    // actualizar sheet/lista de la solicitud msg.id
    if (msg.status === 'pending' || msg.status === 'accepted' || msg.status === 'completed' || msg.status === 'rejected') {
      refreshBalance() // hold → processing → payout, o restore
    }
    if (msg.status === 'completed') showProof(msg.proof_url)
    if (msg.status === 'rejected') showReason(msg.rejection_reason)
  }
}
```

Si no hay WS activo, el detalle sigue en `GET /users/balance/refund-requests/:id`.

---

## 8. CRM Staff — Devoluciones

Pantalla **Devoluciones** en el CRM (`/dashboard/refunds`).

### 8.1 WebSocket de cola

**URL:** `ws(s)://…/users/staff/refund-requests`  
**Rol:** `Staff`

Al conectar:

1. Snapshot de solicitudes `pending` + `accepted` (cada una como `refund_request_update`)
2. `{"type":"connected"}`

Cada `refund_request_update` incluye, además de PM destino:

| Campo | Uso CRM |
|---|---|
| `amount_bs` | **Monto en Bs a pagar por Pago Móvil** (fuente de verdad; es lo que se valida al completar) |
| `amount_usd` | Equivalente informativo (`amount_bs / bcv_rate`) |
| `bcv_rate` | Tasa BCV usada para el equivalente USD |

Si BCV no está disponible, `amount_usd` / `bcv_rate` pueden omitirse (0 / ausentes). `amount_bs` siempre viene.

### 8.2 Accept / Reject (WS)

```json
{ "type": "accept", "request_id": "…" }
```

```json
{ "type": "reject", "request_id": "…", "reason": "Datos de Pago Móvil incorrectos" }
```

Respuesta OK:

```json
{ "type": "action_ok", "action": "accept", "request_id": "…", "status": "accepted" }
```

- `accept`: `pending` → `accepted` (**sin** movimiento de balance; el débito fue al crear). Card sigue en “por pagar”.
- `reject`: `pending` → `rejected` + crédito `balance_refund_restore` (sacar de la cola).

También llega broadcast `refund_request_update` (staff + cliente).

### 8.3 Complete — HTTP (preferido)

Subir comprobante por imagen (no pegar URL) y validar monto:

`POST /users/staff/refund-requests/:id/complete`

```json
{
  "amount_bs": 2080,
  "proof_image": "data:image/jpeg;base64,/9j/4AAQ..."
}
```

| Campo | Regla |
|---|---|
| `amount_bs` | Debe coincidir con `amount_bs` de la solicitud (± 0.01 Bs). Si no → `422` “el monto ingresado no coincide…” |
| `proof_image` | Data-URL / base64 de imagen; el API la sube a Cloudinary |

UX del modal CRM:

1. Mostrar **`amount_bs`** (a transferir) + `amount_usd` informativo + datos PM destino.
2. Input de monto Bs: **verde** si coincide con `amount_bs`, **rojo** si no.
3. Upload de imagen del comprobante (JPG/PNG/WEBP, máx. 5 MB).
4. Confirmar → `POST …/complete` → sacar card de la cola.

Errores útiles del complete:

| Status | Mensaje / causa |
|---|---|
| `400` | Falta comprobante / payload inválido |
| `422` | Monto no coincide / estado no es `accepted` |
| `422` | Error subiendo imagen |

(Opcional) El mismo complete también existe por WS con `type: "complete"` + `amount_bs` + `proof_image`, pero el CRM usa HTTP por el tamaño de la imagen.

---

## 9. Checklist de implementación

### App cliente
- [ ] Pantalla / sheet de balance con `GET /users/balance`
- [ ] Countdown con `refund_eligible_until` cuando `can_request_refund`
- [ ] Flujo: CTA → (registrar PM si falta) → `POST refund-requests`
- [ ] En `order_update`, leer `details.payment` y ofrecer el mismo CTA
- [ ] En el **mismo** WS `orders/manager`, escuchar `type === "refund_request_update"`
- [ ] UI por status: pending (refrescar balance) / accepted / completed (`proof_url`) / rejected (refrescar balance + razón)
- [ ] Listado / detalle HTTP; en historial, `hold` / `processing` / `payout` / `restore` → abrir por `refund_request_id`

### CRM staff
- [x] WS `/users/staff/refund-requests` (distinto de `/orders/staff/reviews`)
- [x] Cola pending + accepted
- [x] Accept / reject por WS
- [x] Complete por HTTP: imagen + monto (verde/rojo)
- [x] `accepted` no cierra el caso hasta complete

---

## 10. Errores / códigos

| Código | Cuándo |
|---|---|
| 401 | Sin sesión |
| 403 | Rol incorrecto |
| 404 | PM no registrado (`GET`), refund/tx ajeno |
| 409 | Solicitud duplicada para el mismo crédito |
| 422 | No elegible, PM requerido al crear, monto de complete no coincide, estado inválido |
| 400 | Falta `proof_image` / payload inválido en complete |

---

## 11. Relación con otros docs

- Payment review (comprobantes, quotes, rechazo local): [`frontend-payment-review.md`](./frontend-payment-review.md)
- Este doc: **solo** balance + devolución en efectivo + sockets (`order_update` CTA + `refund_request_update`).

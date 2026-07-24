# Guía Frontend — Validación de pagos manuales (Pago Móvil)

Documento de contrato para la app del **cliente** (React Native / Expo).  
Describe qué ocurre en cada situación, qué llega por WebSocket y cómo debe reaccionar la UI.

> **Importante:** ya no existe el flujo de “esperar diferencia” ni subir un segundo comprobante sobre la misma orden. Si el pago es incompleto, la orden se **rechaza** y el dinero pagado pasa al **balance del cliente**. Si quiere pedir de nuevo, arma una **nueva orden**.

---

## 0. Precio congelado por el servidor (Quote)

> **Implementado.** El quote se guarda en **Redis** (`order:quote:{id}`, TTL = 30m pago + 15m gracia). Un solo uso (`order:quote:used:{id}`).

### Problema que resuelve

Hoy el front manda `expected_total_price` y, si el descuento del local se acaba entre “ver el carrito” y `POST /orders`, el backend responde **409** y **no crea la orden**.  
Si el cliente **ya hizo el Pago Móvil** de lo que vio en pantalla, ese comprobante **nunca llega al staff**: no hay orden ni `payment_review`.

Además, **no queremos confiar** en un total inventado por el front como fuente de verdad del cobro.

### Solución: quote emitido por el backend

El monto a pagar lo calcula y congela **solo el servidor**, con un TTL. El front solo muestra ese número y luego confirma con el `quote_id`.

```text
Carrito listo
    ↓
POST /orders/quote   ← server calcula total (descuentos, promo, delivery, BCV)
    ↓
Respuesta: quote_id + total_usd + total_bs + breakdown + expires_at
    ↓
UI: “Transfiere exactamente X Bs” (y opcionalmente cuánto usar de balance)
    ↓
Cliente paga / elige balance
    ↓
POST /orders { quote_id, balance_amount_usd?, payment_proof_image? }
    ↓
Server usa el total del quote (no el del front)
```

### Contrato propuesto

#### 1) Crear cotización

`POST /orders/quote` (rol Customer)

Request (misma forma de items/address que CreateOrder, **sin** comprobante):

```json
{
  "shop_id": "uuid",
  "address": {
    "address": "…",
    "latitude": 10.48,
    "longitude": -66.87,
    "instructions": "…"
  },
  "items": [
    {
      "item_id": "uuid",
      "amount": 2,
      "observation": "",
      "flavors": {},
      "additions": []
    }
  ]
}
```

Response `201`:

```json
{
  "quote_id": "uuid",
  "total_usd": 8.0,
  "total_bs": 5895.47,
  "delivery_fee_usd": 1.5,
  "delivery_fee_bs": 1105.4,
  "bcv_rate": 736.93,
  "is_launch_promo": false,
  "expires_at": "2026-07-17T21:10:00Z",
  "proof_grace_until": "2026-07-17T21:25:00Z",
  "items": []
}
```

Reglas del quote:

- Calculado 100% en servidor (incluye descuento del local y promo de lanzamiento vigentes **en ese instante**).
- **TTL de pago** (`expires_at`): **30 minutos** — ventana para transferir / usar balance.
- **Gracia de comprobante** (`proof_grace_until`): **+15 minutos** después de `expires_at`. Sirve solo si el create ya trae `payment_proof_image`.
- Si el descuento se acaba **después** de emitir el quote, se **sigue respetando** el monto congelado del quote (mientras el quote sea aceptable según la tabla de abajo).
- Un quote es de **un solo uso** (al crear la orden se marca consumido).
- **Reutilización:** si el cliente vuelve a pedir quote con el **mismo carrito** (shop + address + items) y el quote activo aún no venció (`expires_at`), el servidor **devuelve el mismo `quote_id`** (mismo monto y countdown). Si cambió el carrito o ya expiró, crea uno nuevo y descarta el anterior.

#### ¿Y si tarda en cargar el comprobante y el quote ya venció?

Caso típico: pidió quote → vio el monto → hizo el Pago Móvil a tiempo → se demoró eligiendo/subiendo la foto → al `POST /orders` el `expires_at` ya pasó.

| Situación al hacer `POST /orders` | ¿Se acepta? | Qué pasa |
|---|---|---|
| Quote vigente (`now < expires_at`), no consumido | Sí | Crea orden con total del quote |
| Quote pasado de `expires_at` pero dentro de la **gracia** y trae `payment_proof_image` | Sí | Crea orden igual, **honrando el total congelado del quote** (no recalcula). Evita comprobante huérfano |
| Quote pasado de `expires_at`, **sin** comprobante (solo balance / reintento) | No | Pedir quote nuevo |
| Quote fuera de la gracia / consumido / de otro user | No | Error; si ya pagó → UI de soporte + guardar comprobante |
| Items del quote ya no existen / local cerrado, etc. | No | Error de catálogo; no inventar otro total |

Resumen para producto:

1. **Antes de pagar en el banco:** el quote debe estar vigente (`expires_at`). Si venció, **nuevo quote** y mostrar el monto nuevo (puede haber cambiado).
2. **Después de pagar, al subir la foto:** si se pasó un poco de `expires_at`, la **gracia** permite crear la orden con el mismo monto congelado, porque el riesgo real es perder el comprobante, no un descuento que ya se “cerró” al emitir el quote.
3. El front debería:
   - Mostrar countdown hasta `expires_at`.
   - Si vence **antes** de transferir → bloquear “Ya pagué” y pedir quote nuevo.
   - Si vence **mientras** comprime/sube la imagen → igual mandar `POST /orders` con ese `quote_id` + proof; el backend aplica gracia.

#### 2) Crear orden con quote

`POST /orders` (rol Customer) — contrato objetivo:

```json
{
  "id": "uuid-orden",
  "quote_id": "uuid",
  "method_payment": "PagoMovil",
  "balance_amount_usd": 4.0,
  "payment_proof_image": "data:image/jpeg;base64,…"
}
```

| Campo | Rol |
|---|---|
| `quote_id` | Obligatorio (fuente de verdad del total) |
| `balance_amount_usd` | Opcional; min $1, max `min(balance, quote.total_usd)` |
| `payment_proof_image` | Obligatorio si el resto Pago Móvil &gt; 0; además habilita la **gracia** si el TTL ya pasó |
| `expected_total_price` | **Deprecado** cuando exista quote; no se usa para fijar el cobro |

El backend:

1. Carga el quote (customer dueño, no consumido).
2. Valida vigencia: `now < expires_at` **o** (`now < proof_grace_until` **y** hay comprobante).
3. Cobra / revisa contra `quote.total_usd` / `quote.total_bs` (congelados).
4. Marca el quote como consumido (un solo uso).
5. Crea la orden (`Payment Review` o `Pending` si 100% balance).

### Qué debe hacer el front

1. Antes de mostrar datos de Pago Móvil / pedir transferencia → llamar `POST /orders/quote`.
2. Mostrar **solo** `total_bs` / `total_usd` del quote + countdown de `expires_at`.
3. Si vence **antes de pagar** → nuevo quote; **no** transferir el monto viejo.
4. Al confirmar: mandar `quote_id` + comprobante (y balance si aplica) lo antes posible.
5. Si al subir la foto el TTL ya pasó pero aún está en gracia → el create debe funcionar igual.
6. Si la API rechaza por quote fuera de gracia / consumido y el usuario **ya pagó** → soporte + “guarda tu comprobante” (caso raro si respetan el countdown).

### Estado actual vs objetivo

| Hoy | Objetivo |
|---|---|
| `expected_total_price` del front + posible **409** si cambió el precio | `quote_id` del server; total congelado en backend |
| Pago hecho → 409 → comprobante huérfano | Pago contra quote vigente → siempre se crea orden/review |

> **Nota:** el path recomendado es quote → create con `quote_id`. Sin `quote_id` sigue funcionando el checkout legacy con `expected_total_price` (sección 0.1), con riesgo de 409 si cambió el precio.

---

## 0.1 Checkout actual: pagos mixtos (balance + Pago Móvil)

`POST /orders` acepta opcionalmente:

```json
{
  "balance_amount_usd": 4.0,
  "payment_proof_image": "data:image/jpeg;base64,…",
  "expected_total_price": 8.0
}
```

| `balance_amount_usd` | Comprobante | Status inicial |
|---|---|---|
| omitido o `0` | Obligatorio | `Payment Review` |
| `>= 1` y `< total` | Obligatorio (solo el resto) | `Payment Review` |
| `== total` (hasta `min(balance, total)`) | No se envía | `Pending` (va al local) |

Reglas:

- Mínimo a usar de balance: **$1**. Máximo: **`min(balance_disponible, total)`**.
- Consultar saldo antes: `GET /users/balance`.
- En `details.payment` quedan `balance_amount_usd` y `pago_movil_amount_usd`.
- Staff valida solo el **resto** en Bs (`expected_amount_bs` del comprobante).
- `expected_total_price` hoy solo detecta carrito desfasado (**409**); **no** es un precio confiable para honrar pagos ya hechos (ver sección 0).

Reembolsos si algo falla después:

| Caso | Qué vuelve al balance |
|---|---|
| Staff `rejected_invalid` | Solo `balance_amount_usd` |
| Staff `rejected_underpaid` | `balance_amount_usd` + lo observado del PM |
| Local `refunded_shop_reject` | `total_price` (recupera todo) |

### Devolución en efectivo (ventana 15 min)

Solo la **porción Pago Móvil** de un crédito puede pedirse como devolución (no lo que ya era balance). `can_request_refund` indica elegibilidad por ventana/monto/saldo (no exige PM registrado). El Pago Móvil se valida al crear: si no hay, el front pide registrarlo (`PUT /users/pago-movil`) y luego `POST /users/balance/refund-requests`.

**Dos vías para el CTA de devolución:**

1. Desde el `order_update` WS: leer `details.payment.transaction_id` + `can_request_refund` + `refund_eligible_until` (overpay / underpaid / shop reject).
2. Desde `GET /users/balance`: mismo `id` del movimiento en el historial.

La ventana de **15 minutos** aplica solo para **crear** la solicitud (`can_request_refund`). Una vez creada:

- `pending` y `accepted` siguen **abiertas** (CRM + alerta Discord cada 5 min).
- Solo terminan con `rejected` o `completed` (pago hecho + comprobante).
- `accepted` **no** cierra el caso: si staff tarda en pagar, seguimos alertando.

| Origen | Cash-refundable |
|---|---|
| `payment_overage` | todo el crédito |
| `payment_partial_credit` | todo el crédito PM |
| `order_rejected_refund` | solo `pago_movil_amount_usd` (0 si 100% balance → no elegible) |

Flujo: `POST /users/balance/refund-requests` (debita como `balance_refund_hold`) → staff WS `accept` (`hold`→`processing`) → `complete` + imagen (`processing`→`payout`) | `reject` (restaura con `balance_refund_restore`). El cliente abre el detalle con `refund_request_id` desde el movimiento de reserva.

Al crear la solicitud (y cada **5 minutos** mientras siga `pending` **o** `accepted`) la API publica `balance_refund_pending` al Redis Stream con `request_id` + `minutes_pending`; el consumer carga cliente/monto/estados desde BD y notifica a Discord.

---

## 1. Flujo general (cuando hay comprobante)

```text
Cliente crea orden (+ opcional balance) + comprobante del resto
        ↓
Orden nace en "Payment Review"
        ↓
Staff valida en CRM (monto del resto PM + últimos 4 dígitos del comprobante)
        ↓
┌───────────────────┬────────────────────┬──────────────────────┬─────────────────────┐
│ Monto correcto    │ Pagó de más        │ Pagó de menos        │ Comprobante inválido│
│ → Pending         │ → Pending          │ → Rejected           │ → Rejected         │
│ → va al local     │ → exceso a balance │ → PM+balance a bal.  │ → solo balance used │
└───────────────────┴────────────────────┴──────────────────────┴─────────────────────┘
```

### Staff CRM — `reference_last4`

En `approve` / `approve_overpay` / `reject_underpaid` el staff **debe** enviar los últimos 4 dígitos del comprobante (`reference_last4`, exactamente `[0-9]{4}`).

- Si esos dígitos **ya existen** en otra review consumida (`approved` | `overpaid` | `rejected_underpaid`), el backend **auto-rechaza** la orden como `rejected_invalid` (motivo: `comprobante duplicado (referencia XXXX)`), sin acreditar el Pago Móvil.
- En `reject_invalid` (foto que no es un pago) **no** se piden los 4 dígitos.
- Respuesta WS al staff: `action_ok` con `action: "reject_invalid"`, `duplicate_reject: true` y `rejection_reason`.

---

## 2. WebSocket del cliente

- **URL:** `ws(s)://<API>/orders/manager`
- **Auth:** cookie de sesión `encargalo-session` (mismo login de customer)
- **Al conectar:** el servidor envía snapshot de órdenes activas + `{"type":"connected"}`
- **Mensaje relevante:** `order_update`

### Forma del mensaje

```json
{
  "type": "order_update",
  "order": {
    "id": "uuid",
    "shop_id": "uuid",
    "customer_id": "uuid",
    "number": 42,
    "status": "Payment Review | Pending | Rejected | …",
    "method_payment": "PagoMovil",
    "total_price": 18.5,
    "total_price_bs": 1850.0,
    "delivery_fee": 1.5,
    "delivery_fee_bs": 150.0,
    "details": {
      "customer": { "…" },
      "shop": { "…" },
      "address": { "…" },
      "items": [ "…" ],
      "payment": {
        "payment_proof_url": "https://…",
        "status": "pending_review | approved | overpaid | rejected_underpaid | rejected_invalid | refunded_shop_reject",
        "expected_amount_bs": 1850.0,
        "observed_amount_bs": 1600.0,
        "difference_bs": 250.0,
        "rejection_reason": "texto opcional",
        "credited_usd": 2.5,
        "balance_amount_usd": 4.0,
        "pago_movil_amount_usd": 4.5,
        "transaction_id": "uuid-del-movimiento-cash-refundable",
        "refundable_pm_usd": 2.5,
        "can_request_refund": true,
        "refund_eligible_until": "2026-07-18T19:28:36Z"
      }
    }
  }
}
```

### Cómo leer el resultado del pago

| Campo | Dónde mirar | Para qué |
|---|---|---|
| Estado de la orden | `order.status` | Flujo de la orden (activa / rechazada / en local) |
| Resultado de la revisión | `order.details.payment.status` | Motivo exacto del resultado del comprobante |
| Montos de la revisión | `order.details.payment.*` | Explicar al usuario cuánto faltó / sobró / se acreditó |

> `details.payment` puede venir incompleto al crear la orden (solo `payment_proof_url`).  
> Tras la decisión del staff, el backend **sincroniza** todos los campos relevantes y vuelve a emitir `order_update`.

---

## 3. Situaciones (qué hacer en UI)

### A) Orden creada — validando comprobante

| | |
|---|---|
| `order.status` | `"Payment Review"` |
| `details.payment.status` | normalmente ausente o aún no resuelto |
| Qué ve el cliente | “Estamos validando tu comprobante de pago” |
| Acciones | **No** pedir otro comprobante. Solo esperar. |
| Local | **No** ve la orden todavía |

---

### B) Monto correcto — aceptado

| | |
|---|---|
| `order.status` | `"Pending"` (luego sigue el flujo normal del local) |
| `details.payment.status` | `"approved"` (si está sincronizado) |
| Qué ve el cliente | “Pago confirmado. Tu pedido fue enviado al local.” |
| Balance | No cambia |
| Acciones | Continuar tracking normal (`Pending` → preparación → etc.) |

---

### C) Pagó de más — aceptado + crédito

| | |
|---|---|
| `order.status` | `"Pending"` |
| `details.payment.status` | `"overpaid"` |
| Campos clave | `expected_amount_bs`, `observed_amount_bs`, `difference_bs`, `credited_usd`, `transaction_id`, `refundable_pm_usd`, `can_request_refund`, `refund_eligible_until` |
| Qué ve el cliente | “Pago confirmado. Detectamos un exceso de X Bs; te acreditamos Y USD a tu balance.” |
| Balance | Sube `credited_usd` (`movement_type: payment_overage`) |
| Acciones | Continuar tracking. Si `can_request_refund`: CTA devolución con `transaction_id` (o vía historial de balance). |

**Ejemplo de lectura:**

```ts
const p = order.details?.payment
if (order.status === 'Pending' && p?.status === 'overpaid') {
  // p.difference_bs = exceso en Bs
  // p.credited_usd  = lo acreditado al balance
}
```

---

### D) Pagó de menos — rechazado + crédito de lo pagado

| | |
|---|---|
| `order.status` | `"Rejected"` |
| `details.payment.status` | `"rejected_underpaid"` |
| Campos clave | `expected_amount_bs`, `observed_amount_bs`, `difference_bs`, `credited_usd`, `transaction_id` (crédito PM), `refundable_pm_usd`, `can_request_refund`, `refund_eligible_until` |
| Qué ve el cliente | “Tu pedido fue rechazado porque el pago estaba incompleto. Lo que pagaste (X Bs ≈ Y USD) quedó en tu balance.” |
| Balance | Sube `credited_usd` con **todo lo observado** (`movement_type: payment_partial_credit`) |
| Acciones | **No** pedir segundo comprobante en esta orden. Si `can_request_refund`: CTA devolución. También “Hacer un nuevo pedido” / “Usar mi balance”. |

**Reglas de producto (importante):**

1. Esta orden **muere** (`Rejected`). No se reabre.
2. El dinero no se pierde: queda en balance del cliente (USD).
3. Si quiere el pedido, arma una **nueva orden**.
4. Cómo combinar balance + nuevo Pago Móvil es decisión de UI/producto (el backend aún cobra con comprobante en `CreateOrder`; el uso de balance como medio de pago puede venir en una iteración siguiente — hoy al menos deben **mostrar** el balance y explicar que ese dinero ya está ahí).

**Ejemplo de lectura:**

```ts
const p = order.details?.payment
if (order.status === 'Rejected' && p?.status === 'rejected_underpaid') {
  // p.expected_amount_bs  = lo que debía pagar
  // p.observed_amount_bs  = lo que efectivamente llegó
  // p.difference_bs       = lo que faltó (expected - observed)
  // p.credited_usd        = lo acreditado al balance (conversión de observed)
}
```

---

### E) Comprobante inválido — rechazado sin crédito

| | |
|---|---|
| `order.status` | `"Rejected"` |
| `details.payment.status` | `"rejected_invalid"` |
| Campos clave | `rejection_reason` |
| Qué ve el cliente | “Tu pedido fue rechazado: {rejection_reason}” |
| Balance | **No** cambia |
| Acciones | CTA: “Hacer un nuevo pedido” (y cargar un comprobante válido) |

---

### F) Local rechaza la orden — reembolso al balance

Ocurre **después** de que la orden ya estaba en el local (`Pending`).

Incluye dos disparadores con el **mismo** payload para el front:

1. El local rechaza manualmente (WS manager).
2. **Auto-timeout:** la orden lleva **10 minutos** en `Pending` sin aceptación. Motivo típico: local cerrado o fuera de la app.

| | |
|---|---|
| `order.status` | `"Rejected"` |
| `details.payment.status` | `"refunded_shop_reject"` |
| Campos clave | `credited_usd` (= `order.total_price`), `rejection_reason`, y si hay porción PM: `transaction_id`, `refundable_pm_usd`, `can_request_refund`, `refund_eligible_until` |
| Qué ve el cliente | “El local rechazó tu pedido. Te acreditamos $X a tu balance.” (o el `rejection_reason` si lo muestran) |
| Balance | Sube `order.total_price` USD (`movement_type: order_rejected_refund`) |
| Nota | Si antes hubo overpay, el exceso **ya** estaba en balance; este crédito es el total de la orden (lo que se “cobró” por el pedido). En 100% balance, esto **devuelve** lo debitado al crear. |
| Acciones | CTA devolución si `can_request_refund` (usar `transaction_id` del mismo `details.payment`); también vía `GET /users/balance`. Alternativa: “Hacer un nuevo pedido” |

El reloj del auto-timeout arranca cuando la orden **entra a `Pending`** (al crear si es 100% balance, o cuando staff aprueba el comprobante). Mientras esté en `Payment Review`, no corre.

### Aviso Discord (retraso del local)

Mientras la orden sigue en `Pending` sin que el local acepte, la API publica `order_pending_unassigned` en los minutos **4** y **8** (con `minutes_pending`). El consumer arma el embed (local, #orden, owner, dirección, subtotal) e incluye el texto del tiempo (“4 minutos”, “8 minutos”).

A los **10 minutos** corre el auto-timeout: rechazo como rechazo del local (`refunded_shop_reject`).

- Fuera de `production` → canal Discord de desarrollo (`DISCORD_DEV_CHANNEL_ID`).
- En `production` → canal dedicado (`DISCORD_ORDERS_PENDING_UNASSIGNED_ID`).

---

## 4. Cómo distinguir los tipos de rechazo

Todos dejan `order.status === "Rejected"`. **No basta con el status de la orden.**  
Usen siempre `details.payment.status`:

| `details.payment.status` | Significado | ¿Hay crédito? |
|---|---|---|
| `rejected_underpaid` | Pago incompleto (staff) | Sí → lo observado (`payment_partial_credit`) |
| `rejected_invalid` | Comprobante no válido (staff) | No |
| `refunded_shop_reject` | Local rechazó el pedido (manual o auto-timeout sin aceptación) | Sí → total de la orden (`order_rejected_refund`) |

Si por algún motivo no llega `details.payment.status`, fallback:

- Si hay `credited_usd` + montos observados → tratar como underpaid.
- Si hay `rejection_reason` y no hay crédito → inválido.

---

## 5. Crear orden (recordatorio de contrato)

### Hoy (sin quote)

`POST /orders` (rol Customer)

```json
{
  "id": "uuid-generado-en-front",
  "shop_id": "uuid",
  "method_payment": "PagoMovil",
  "payment_proof_image": "data:image/jpeg;base64,…",
  "expected_total_price": 18.5,
  "balance_amount_usd": 0,
  "address": { "address": "…", "latitude": 0, "longitude": 0 },
  "items": [ /* … */ ]
}
```

Notas:

- `expected_total_price` es **top-level**. Si el backend recalcula distinto → **409** (riesgo de comprobante huérfano; ver sección 0).
- `payment_proof_image` es obligatorio si hay resto Pago Móvil.
- Respuesta `201`: suele nacer en `"Payment Review"`; el local no la ve hasta aprobación.

### Objetivo (con quote)

Ver sección 0: primero `POST /orders/quote`, luego `POST /orders` con `quote_id` (sin depender del total del front).

---

## 6. Balance del cliente

`GET /users/balance` (rol Customer)

```json
{
  "balance": 12.5,
  "transactions": [
    {
      "id": "uuid",
      "amount": 2.5,
      "movement_type": "payment_overage | payment_partial_credit",
      "order_id": "uuid",
      "created_at": "2026-07-17T…"
    }
  ]
}
```

| `movement_type` | Origen |
|---|---|
| `payment_overage` | Pagó de más (orden aceptada) |
| `payment_partial_credit` | Pagó de menos (orden rechazada por staff, se acreditó lo del PM) |
| `order_rejected_refund` | Local rechazó la orden (se acreditó el total de la orden) |
| `balance_payment` | Débito al crear orden con balance |
| `balance_payment_refund` | Devolución del balance usado (staff reject / fallo de create) |

Recomendación UI:

- Tras `overpaid` o `rejected_underpaid`, refrescar balance (`GET /users/balance`) o invalidar cache.
- Mostrar saldo disponible en checkout / perfil.

---

## 7. Checklist de implementación (app cliente)

- [ ] Manejar status nuevo `"Payment Review"` en tracking (copy de “validando comprobante”).
- [ ] Escuchar `order_update` y leer `order.details.payment`.
- [ ] Pantalla / modal para `overpaid` (mostrar exceso Bs + crédito USD).
- [ ] Pantalla / modal para `rejected_underpaid` (mostrar faltante + crédito a balance + CTA nueva orden).
- [ ] Pantalla / modal para `rejected_invalid` (mostrar `rejection_reason` + CTA nueva orden).
- [ ] Pantalla / modal para `refunded_shop_reject` (local rechazó + crédito a balance).
- [ ] **No** implementar UI de “subir diferencia” ni timeout de 10 minutos (eliminado del backend).
- [ ] Integrar `GET /users/balance` y mostrar movimientos (incl. `balance_payment` / reembolsos).
- [ ] Al reconectar WS, el snapshot trae órdenes activas; las `Rejected` no vuelven en activas → usar historial / último `order_update` cacheado si necesitan re-mostrar el mensaje.
- [ ] Pedir `POST /orders/quote` antes de mostrar datos de transferencia; pagar solo el `total_bs` del quote; crear orden con `quote_id` (+ countdown `expires_at` / gracia con comprobante).

---

## 8. Qué NO hacer

- No esperar un status `"Awaiting Difference"` (ya no existe).
- No llamar `POST /orders/{id}/payment-proof` (endpoint eliminado).
- No asumir que `Rejected` siempre significa comprobante inválido: mirar `details.payment.status`.
- No inventar textos con montos si faltan campos; preferir fallback genérico y refrescar historial/balance.
- No usar un total “del carrito local” como fuente de verdad del cobro: el monto a transferir debe venir del **quote del servidor** (sección 0).
- No pedir al usuario que pague y después llamar a create order sin un `quote_id` vigente (evita comprobantes huérfanos).

---

## 9. Tabla rápida de decisión UI

| `order.status` | `payment.status` | Título sugerido | CTA principal |
|---|---|---|---|
| `Payment Review` | — / `pending_review` | Validando tu pago | Esperar |
| `Pending` | `approved` | Pago confirmado | Ver pedido |
| `Pending` | `overpaid` | Pago confirmado + crédito | Ver pedido / Ver balance |
| `Rejected` | `rejected_underpaid` | Pago incompleto — dinero en tu balance | Nueva orden |
| `Rejected` | `rejected_invalid` | Comprobante inválido | Nueva orden |
| `Rejected` | `refunded_shop_reject` | Local rechazó — reembolso a tu balance | Nueva orden / Ver balance |

---

## 10. Pseudocódigo sugerido

```ts
function handleOrderUpdate(msg: { type: string; order: Order }) {
  if (msg.type !== 'order_update') return
  const order = msg.order
  const payment = order.details?.payment

  switch (order.status) {
    case 'Payment Review':
      showValidatingPayment(order)
      break

    case 'Pending':
      if (payment?.status === 'overpaid') {
        showAcceptedWithOverage({
          differenceBs: payment.difference_bs,
          creditedUsd: payment.credited_usd,
        })
      } else {
        showAccepted(order)
      }
      break

    case 'Rejected':
      if (payment?.status === 'rejected_underpaid') {
        showRejectedUnderpaid({
          expectedBs: payment.expected_amount_bs,
          observedBs: payment.observed_amount_bs,
          missingBs: payment.difference_bs,
          creditedUsd: payment.credited_usd,
        })
        refreshBalance()
      } else if (payment?.status === 'rejected_invalid') {
        showRejectedInvalid(payment.rejection_reason)
      } else if (payment?.status === 'refunded_shop_reject') {
        showShopRejectedRefund({
          creditedUsd: payment.credited_usd, // = order.total_price
          reason: payment.rejection_reason,
        })
        refreshBalance()
      } else {
        showRejectedGeneric()
      }
      break

    default:
      // flujo normal de tracking
      updateTracking(order)
  }
}
```

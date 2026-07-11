**Promociones de Lanzamiento para Negocios Nuevos**

```
Negocio abre por primera vez
        │
        ▼
 ┌───────────────────────────────┐
 │  Primeras 24 horas            │  50% cliente + comisión 0%
 └───────────────────────────────┘
        │
        ▼
 ┌───────────────────────────────┐
 │  Horas 24 a 192 (8 días)      │  comisión 0%
 └───────────────────────────────┘
        │
        ▼
   Después de la hora 192           reparto normal (90% local / 10% plataforma)
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**¿Cuándo arranca todo?**

En el momento en que el local abre por primera vez en la app (botón "Abrir" del Owner). Ese momento es el punto de partida de las dos promociones. Abrir y cerrar el local después de esa primera vez no reinicia el reloj — la ventana ya quedó fijada.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**1. Primeras 24 horas — Descuento al cliente**

Durante las primeras 24 horas desde la apertura:

- El **cliente paga 50%** del valor de los productos.
- **Encargalo pone el otro 50%** de su bolsillo (subsidio real, sale de la plataforma).
- El **local recibe el 100%** del valor de los productos, como si no hubiera descuento.
- El **domicilio se cobra normal**, sin descuento.

> Quién paga qué: cliente 50% + Encargalo 50% = 100% para el local.

**2. Primeras 192 horas (8 días) — Comisión 0%**

Durante las primeras 192 horas (8 días) desde la apertura:

- Encargalo **no cobra su comisión del 10%** sobre las ventas.
- El **local recibe el 100%** del valor de los productos en cada orden completada.
- A diferencia de las primeras 24h, aquí el cliente paga el precio completo (no hay descuento) — Encargalo simplemente no se queda con su parte.

> Después de la hora 192, el reparto vuelve a ser el normal: local 90% / plataforma 10% (ver `ciclo-vida-orden.md`).

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Ejemplo con números** (orden de $12 en productos, $2 de domicilio)

| Momento | Cliente paga | Local recibe | Encargalo |
|---|---|---|---|
| Primeras 24h | $6 + $2 domicilio = $8 | $12 | Pone $6, no cobra comisión |
| Horas 24-192 | $12 + $2 domicilio = $14 | $12 | No cobra comisión (antes se quedaba $1.20) |
| Después de la hora 192 | $12 + $2 domicilio = $14 | $10.80 (90%) | Se queda $1.20 (10%) |

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Descuentos propios del negocio** *(feature aparte, independiente de lo anterior)*

Cualquier local puede poner su propio descuento a un producto, cuando quiera, sin relación con las promociones de lanzamiento. Ese descuento lo paga el local con su propio margen — Encargalo nunca lo subsidia. Si un local tiene su propio descuento activo durante sus primeras 24 horas, el 50%/50% de la promo de lanzamiento se calcula sobre el precio ya rebajado por el local, para que Encargalo no termine subsidiando un descuento que el local decidió por su cuenta.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Lo que ve el cliente**

El precio con descuento (primeras 24h, y/o el descuento propio del local si tiene uno) se muestra en el catálogo antes de comprar, no es una sorpresa al pagar. Si el precio cambia justo antes de confirmar, el sistema rechaza la orden y le pide al cliente revisar el carrito actualizado — nunca se le cobra algo distinto a lo que vio.

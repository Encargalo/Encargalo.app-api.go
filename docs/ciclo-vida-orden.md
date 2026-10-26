**Ciclo de Vida de una Orden**

```
Pending → InPreparation → Ready → On The Way → Completed
        ↘ Rejected (local)
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**1. Pending**
El local recibe la orden y decide si la acepta o la rechaza.
> Quién actúa: el local (aceptar o rechazar)

**2. InPreparation**
El local acepta la orden y comienza la preparación. Al entrar a este estado ocurren dos cosas automáticamente:
- Se genera un **código de recogida** (pickup code)
- Se notifica a los riders disponibles por WebSocket con los detalles de la carrera
> Quién actúa: el local (marcar como Ready o Rejected)

**3. Ready** *(opcional)*
El local marca la orden como lista para ser recogida. Indica que los productos ya están empacados y esperando al rider.
> Quién actúa: el rider (confirmar recogida con el código)

**4. On The Way**
El rider confirma la recogida usando el **código de recogida**. Al validarlo:
- El código queda marcado como usado
- Se genera automáticamente un **código de entrega** (delivery code)
- La orden pasa a `On The Way`
> Quién actúa: el cliente (confirmar entrega con el código de entrega)

**5. Completed**
El cliente confirma que recibió la orden usando el **código de entrega**. En este momento se registran los pagos:

- **Rider** → 90 % de la tarifa de domicilio
- **Local** → 90 % del total de productos (total − domicilio)
- **Plataforma** → 10 % del domicilio + 10 % de productos

*Ejemplo: orden de $12, domicilio $2 → Rider $1.60 · Local $9.00 · Plataforma $1.40*

**Rejected**
El local rechaza la orden. No genera movimientos financieros.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Códigos de verificación**

- **Pickup code** — se genera al pasar a InPreparation, lo usa el rider para confirmar que recogió la orden
- **Delivery code** — se genera al confirmar la recogida, lo usa el cliente para confirmar que recibió la orden

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Balance**

**Rider**
- Acumula saldo positivo con cada entrega completada (+90 % del domicilio)
- Puede solicitar retiro cuando su saldo supera **$3**, máximo una vez cada **24 horas**
- Al procesar el pago su saldo vuelve a $0

**Local**
- Acumula el 90 % del valor de los productos vendidos por entrega completada
- Puede consultar su balance e historial de movimientos desde la app

**Plataforma**
- Retiene el 10 % del domicilio y el 10 % del valor de productos en cada orden completada

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Método de pago**
El único método aceptado es **Pago Móvil**. El cliente envía la imagen del comprobante de transferencia al crear la orden.

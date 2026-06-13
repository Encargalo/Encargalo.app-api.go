## ADDED Requirements

### Requirement: El monto de carrera se calcula por distancia y tarifa horaria

El sistema SHALL calcular el monto de carrera usando la fórmula:

```
monto_carrera = distancia_km × tarifa_horaria
```

La tarifa se determina según la hora de **creación del pedido** (`created_at`), no de entrega:

| Horario | Tarifa |
|---|---|
| Diurna (antes de las 22:00) | $0.40 / km |
| Nocturna (desde las 22:00) | $0.60 / km |

#### Scenario: Pedido creado en horario diurno

- **WHEN** un pedido con `created_at` antes de las 22:00 se completa con una distancia de 5 km
- **THEN** `monto_carrera` SHALL ser `5 × $0.40 = $2.00`

#### Scenario: Pedido creado en horario nocturno

- **WHEN** un pedido con `created_at` a las 22:00 o después se completa con una distancia de 5 km
- **THEN** `monto_carrera` SHALL ser `5 × $0.60 = $3.00`

#### Scenario: La hora de entrega no afecta la tarifa

- **WHEN** un pedido es creado a las 21:55 pero completado a las 22:10
- **THEN** la tarifa aplicada SHALL ser la diurna ($0.40/km) porque la creación fue antes de las 22:00

---

### Requirement: La distancia se obtiene del pedido

El sistema SHALL usar la distancia en km del campo de la orden para el cálculo. Si la distancia no está disponible, el sistema SHALL rechazar el cálculo y registrar el error.

#### Scenario: Distancia disponible en el pedido

- **WHEN** el pedido tiene una distancia en km válida (> 0)
- **THEN** el sistema SHALL calcular `monto_carrera` correctamente

#### Scenario: Distancia no disponible

- **WHEN** el pedido no tiene distancia en km registrada
- **THEN** el sistema SHALL registrar el error en los logs y no ajustar el balance del rider, evitando un movimiento inválido

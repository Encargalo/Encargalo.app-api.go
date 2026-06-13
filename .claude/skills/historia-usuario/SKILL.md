---
name: historia-usuario
description: Generador de Historias de Usuario estructuradas para el backlog de Encargalo.app-api.go, alineadas con la arquitectura y el dominio.
argument-hint: "[descripción del requerimiento o funcionalidad]"
---

# Generador de Historias de Usuario — Encargalo.app-api.go

## Propósito
Generar Historias de Usuario completas, implementables y alineadas con la arquitectura y dominio del proyecto.

## Contexto activo
- `/contexto-proyecto` — módulos, roles, lenguaje ubicuo
- `/reglas-arquitectura` — constraints que afectan la implementación

## Entradas mínimas

Si no están claras, preguntar:
1. ¿Quién es el actor? (`Customer`, `Owner`, `OrderManager`, `Employee`)
2. ¿Qué quiere hacer?
3. ¿Para qué? (objetivo de negocio)
4. ¿Hay restricciones de negocio conocidas?
5. ¿Qué módulo se verá afectado?

## Formato de Historia de Usuario

```
## HU-NNN: [Título descriptivo]

**Como** [rol: Customer | Owner | OrderManager | Employee]
**Quiero** [acción concreta]
**Para** [objetivo de negocio]

### Criterios de aceptación
- [ ] CA-1: [condición verificable — sí/no o valor medible]
- [ ] CA-2: [condición verificable]
- [ ] CA-N: [condición verificable]

### Reglas de negocio
- RN-1: [restricción o invariante del dominio]
- RN-N: [restricción o invariante del dominio]

### Notas técnicas
- Módulo afectado: `internal/<modulo>/`
- Endpoints estimados: [MÉTODO] /ruta
- Autenticación: [requerida / no requerida]
- Rol requerido: [Customer | Owner | OrderManager | Employee | N/A]
- Impacto en BD: [sí/no + descripción breve]
- Impacto en Redis: [sí/no + descripción breve]

### Definition of Ready (DoR)
- [ ] Criterios de aceptación completos y verificables
- [ ] Reglas de negocio definidas
- [ ] Módulo y endpoints estimados identificados
- [ ] Sin ambigüedades bloqueantes

### Estimación sugerida
[XS / S / M / L / XL] — [justificación breve]
```

## Guía de estimación

| Talla | Descripción |
|---|---|
| XS | Un endpoint simple CRUD sin lógica de negocio compleja |
| S | 1-2 endpoints con reglas de negocio simples |
| M | Feature completa (3-4 endpoints, lógica moderada, sin integraciones externas) |
| L | Feature con integración externa, Redis, o lógica de negocio compleja |
| XL | Feature multi-módulo o que requiere migración de esquema significativa |

## Reglas de generación

- NO inventar reglas de negocio — si no están claras, preguntar.
- NO generar código — el output es exclusivamente la HU.
- Los criterios de aceptación DEBEN ser verificables (sí/no o valor medible).
- El resultado será consumido por `/generar-feature` para la implementación.
- Si la HU es demasiado grande (L/XL), proponer split en HUs más pequeñas.
- Usar el lenguaje ubicuo del proyecto (definido en `/contexto-proyecto`) en todo el texto.

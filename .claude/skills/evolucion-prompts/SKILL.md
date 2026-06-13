---
name: evolucion-prompts
description: Gobernanza del sistema de skills de Encargalo.app-api.go. Controla cómo evolucionan los prompts y previene duplicación de reglas.
argument-hint: "[descripción del cambio propuesto]"
---

# Evolución del Sistema de Prompts — Encargalo.app-api.go

## Propósito
Controlar cómo evoluciona el sistema de skills para que permanezca coherente, sin duplicaciones y refleje el estado real del proyecto.

## Mapa de ownership (anti-duplicación)

| Skill | Dueño de |
|---|---|
| `/reglas-arquitectura` | Reglas prescriptivas DEBE/NO DEBE (única fuente) |
| `/arquitecto` | Routing, modos, contrato de interacción |
| `/contexto-proyecto` | Mapa de dominio, módulos, lenguaje ubicuo, stack |
| `/flujo-desarrollo` | Proceso por bloques, orden de capas, gates |
| `/flujo-documentacion` | Qué documentos actualizar y cómo |
| `/generar-feature` | Template operativo para crear features |
| `/auditor-arquitectura` | Checklist de auditoría, clasificación, veredicto |
| `/refactor` | Proceso de refactor guiado |
| `/historia-usuario` | Formato y reglas de Historias de Usuario |
| `/auditor-seguridad` | Checklist OWASP, clasificación de vulnerabilidades |

## Cuándo evolucionar un skill

✅ **Evolucionar cuando**:
- Un hallazgo de auditoría se repite en múltiples bloques (señal de regla faltante).
- Se agrega un nuevo módulo o bounded context al proyecto.
- Cambia una tecnología del stack (ORM, framework, DI).
- Se detecta una convención implícita que debería ser explícita.
- El asistente genera código que viola una regla, y la regla no está escrita.

❌ **NO evolucionar cuando**:
- Es una excepción puntual de una sola feature.
- La regla ya existe en otro skill (sería duplicación).
- El cambio es cosmético sin impacto en el comportamiento del asistente.

## Proceso de cambio

### Paso 1 — Identificar el problema
- Describir el problema con evidencia: ¿qué generó el asistente mal?
- Identificar cuántas veces se ha repetido.

### Paso 2 — Determinar ownership

Usar la tabla de ownership:
- ¿Es una regla DEBE/NO DEBE? → `/reglas-arquitectura`
- ¿Es sobre cómo trabajar en bloques? → `/flujo-desarrollo`
- ¿Es sobre el dominio, módulos o stack? → `/contexto-proyecto`
- ¿Es sobre documentación? → `/flujo-documentacion`
- ¿Es sobre routing o modos? → `/arquitecto`

### Paso 3 — Verificar no duplicación
Antes de escribir, confirmar que:
- La regla no existe ya en el skill objetivo.
- La regla no existe en otro skill (que sería el lugar incorrecto).
- El cambio no contradice ninguna regla existente.

### Paso 4 — Aplicar cambio mínimo
- Modificar únicamente el skill dueño.
- Cambio mínimo: agregar o modificar la regla específica, no reescribir el skill.
- Si hay contradicción con otra regla, resolverla explícitamente.

### Paso 5 — Documentar el cambio
Registrar al final del skill modificado:

```
## Historial de cambios
- [FECHA]: [Cambio aplicado] — Motivo: [evidencia del problema]
```

## Regla de no-duplicación

Las reglas DEBE/NO DEBE viven **exclusivamente** en `/reglas-arquitectura`.
Los demás skills referencian esas reglas, nunca las duplican.
Si un skill dice "DEBE hacer X" y `/reglas-arquitectura` ya lo dice, el skill debe referenciarlo en su lugar.

---
name: evolucion-prompts
description: Gobernanza del sistema de skills de Encargalo.app-api.go. Controla cómo evolucionan los prompts y previene duplicación de reglas.
argument-hint: "[descripción del cambio propuesto]"
---

# Evolución del Sistema de Prompts — Encargalo.app-api.go

## Propósito
Controlar cómo evoluciona el sistema de skills para que permanezca coherente, sin duplicaciones y refleje el estado real del proyecto.

## Mapa de ownership (anti-duplicación)

### Skills del proyecto (`.claude/skills/`)

| Skill | Dueño de |
|---|---|
| `/reglas-arquitectura` | Reglas prescriptivas DEBE/NO DEBE (única fuente) |
| `/arquitecto` | Routing, modos, contrato de interacción |
| `/contexto-proyecto` | Mapa de dominio, módulos, lenguaje ubicuo, stack |
| `/flujo-desarrollo` | Proceso por bloques, orden de capas, gates |
| `/flujo-documentacion` | Qué documentos actualizar y cómo |
| `/generar-feature` | Template operativo para crear features + bridge con OpenSpec |
| `/auditor-arquitectura` | Checklist de auditoría, clasificación, veredicto |
| `/refactor` | Proceso de refactor guiado |
| `/historia-usuario` | Formato y reglas de Historias de Usuario |
| `/auditor-seguridad` | Checklist OWASP, clasificación de vulnerabilidades |

### Skills de OpenSpec (generados por `openspec init`, NO editar manualmente)

| Skill | Dueño de |
|---|---|
| `/openspec-propose` | Crear un change con todos sus artefactos en un paso |
| `/openspec-apply-change` | Implementar tasks de un change (usa `/generar-feature` como destino) |
| `/openspec-archive-change` | Archivar un change completado |
| `/openspec-explore` | Modo exploración / thinking partner antes de proponer |
| `/openspec-sync-specs` | Sincronizar specs delta de un change a specs principales |

### Archivo de configuración OpenSpec

| Archivo | Dueño de |
|---|---|
| `openspec/config.yaml` | Contexto del proyecto para los artefactos OpenSpec: stack, lenguaje ubicuo, reglas de specs/tasks/design |

> **Regla**: `openspec/config.yaml` y `/contexto-proyecto` tienen solapamiento intencionado. El primero guía a OpenSpec al *crear artefactos*. El segundo guía al asistente al *implementar*. No son duplicación — tienen audiencias distintas.

## Relación entre sistemas

```
/opsx:propose  ──→  openspec/changes/<nombre>/
                         proposal.md
                         specs/
                         design.md
                         tasks.md
                              │
                              ▼
              /generar-feature  (lee los artefactos, omite preguntas ya respondidas)
                              │
                              ▼
              Implementación por bloques hexagonales
                              │
                              ▼
              /opsx:archive  (archiva el change completo)
```

Los skills de OpenSpec se actualizan con `openspec update` (no editar manualmente).
Los skills del proyecto se evolucionan siguiendo el proceso de este skill.

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

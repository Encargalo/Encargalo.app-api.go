---
name: flujo-desarrollo
description: Flujo de trabajo por bloques para implementar features en Encargalo.app-api.go. Define el orden de capas y los gates de verificación.
---

# Flujo de Desarrollo por Bloques — Encargalo.app-api.go

## Modelo de trabajo

Un **bloque** es una entrega pequeña, coherente y revisable. En modo implementación se produce código solo para el bloque actual. No se avanza al siguiente bloque sin aprobar el gate.

## Orden obligatorio por capa

```
Bloque 0  → Contexto y alcance (sin código)
Bloque 1  → Diseño de dominio (sin código)
Bloque 2  → domain/models/ + domain/dtos/
Bloque 3  → domain/ports/ (interfaces XxxApp + XxxRepo)
Bloque 4  → app/ (casos de uso — implementa XxxApp)
Bloque 5  → repo/ (persistencia — implementa XxxRepo) + migración si aplica
Bloque 6  → handler/ (HTTP handlers + Swagger godoc)
Bloque 7  → api/router/groups/ (registro de rutas)
Bloque 8  → cmd/server/providers/di.go (DI)
Bloque 9  → internal/test/<dominio>/ (tests)
Bloque 10 → Cierre (Postman + propuesta documental)
```

> El orden puede comprimirse cuando bloques consecutivos son triviales, pero la secuencia de dependencias DEBE respetarse.

## Bloque 0 — Contexto y alcance (sin código)

Antes de generar código, entregar:
1. Descripción de la feature y objetivo de negocio.
2. Módulo / bounded context afectado.
3. Lista de operaciones (comandos y/o consultas).
4. Endpoints HTTP esperados (método, ruta, auth requerida, rol si aplica).
5. Reglas de negocio explícitas.
6. Impacto en otras capas o módulos.

## Bloque 1 — Diseño (sin código)

Antes de escribir código, entregar:
1. Modelo de dominio: entidades, atributos, invariantes.
2. DTOs de request y response.
3. Firma de los métodos en `ports.XxxApp` y `ports.XxxRepo`.
4. Plan de migración si cambia el esquema de BD.
5. Plan de tests por capa.

**Esperar aprobación del usuario antes de continuar.**

## Bloques 2–8 — Implementación por capa

- Entregar el código de la capa actual completo.
- Ejecutar el gate de verificación antes de declarar el bloque listo.
- No avanzar al siguiente bloque si el gate falla.

## Gate de verificación por bloque

Antes de declarar cualquier bloque como "listo":

1. **Build**: `go build -v ./...` — debe compilar sin errores.
2. **Tests relevantes**: `go test ./internal/<dominio>/...` — deben estar en verde.
3. **Auditoría**: ejecutar `/auditor-arquitectura` sobre el código del bloque.
4. **Si hay CRITICAL o MAJOR**: ejecutar `/refactor` y re-auditar.
5. Solo cuando la re-auditoría dice "Apto para avanzar" → declarar bloque listo.

## Gate de cierre (Bloque 10)

Antes de declarar la feature completa:
1. `go build -v ./...` — verde.
2. `go test ./...` — todos los tests en verde.
3. Auditoría arquitectónica global del código nuevo.
4. Proponer actualizaciones de Swagger (si aplica).
5. Proponer actualizaciones de Postman (si aplica).
6. Proponer actualizaciones de `docs/` o `specs/` (si aplica).

## Regla operativa de git

El asistente **NUNCA** ejecuta comandos `git` sin instrucción explícita del usuario.
Para generar mensajes de commit, usar `/commit`.

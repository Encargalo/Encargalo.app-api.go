---
name: auditor-arquitectura
description: Auditor arquitectónico para Encargalo.app-api.go. Detecta violaciones a las reglas hexagonales y emite veredicto de bloque.
argument-hint: "[ruta o módulo a auditar]"
---

# Auditor de Arquitectura — Encargalo.app-api.go

## Contexto activo
Valida contra: `/reglas-arquitectura`
Conoce el dominio desde: `/contexto-proyecto`

## Clasificación de hallazgos

| Severidad | Descripción |
|---|---|
| **CRITICAL** | Violación grave que rompe la arquitectura hexagonal o introduce deuda irrecuperable |
| **MAJOR** | Problema de diseño que viola un constraint DEBE/NO DEBE o genera deuda técnica significativa |
| **MINOR** | Problema de calidad, consistencia o convención que no bloquea pero debe corregirse |
| **INFO** | Observación sin impacto directo — mejora opcional |

## Checklist de auditoría

### Capa de Dominio (`domain/`)
- [ ] `models/` no importa nada de `handler/`, `app/`, `repo/`
- [ ] `ports/` solo contiene interfaces `XxxApp` y `XxxRepo`
- [ ] `dtos/` solo contiene structs — sin lógica de negocio ni dependencias de infraestructura
- [ ] Los modelos no tienen dependencias de infraestructura

### Capa de Aplicación (`app/`)
- [ ] Implementa la interfaz `ports.XxxApp` completa
- [ ] Solo depende de `ports.XxxRepo` (interface), no del struct de repo
- [ ] No accede directamente a base de datos ni Redis
- [ ] No devuelve errores HTTP — solo errores de dominio (`errcustom`)
- [ ] Valida invariantes de negocio
- [ ] No inicia transacciones directamente

### Capa de Repositorio (`repo/`)
- [ ] Implementa la interfaz `ports.XxxRepo` completa
- [ ] Usa bun ORM para acceso a datos
- [ ] Las transacciones se manejan aquí (no en `app/`)
- [ ] No contiene lógica de negocio

### Capa HTTP (`handler/`)
- [ ] Solo depende de `ports.XxxApp` (interface), no del struct de app
- [ ] Flujo: bind request → call app → map errors → return response
- [ ] No contiene lógica de negocio
- [ ] Mapea `errcustom` a `echo.NewHTTPError` correctamente
- [ ] Tiene anotaciones Swagger completas (`@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`)
- [ ] Sesión obtenida con `ctx.Value(models.SessionKey).(models.ActiveSession)`
- [ ] Middlewares de auth correctamente encadenados

### Routing y DI
- [ ] Rutas registradas en `api/router/groups/<modulo>.go`
- [ ] Dependencias registradas en `cmd/server/providers/di.go`
- [ ] No hay DI manual fuera del contenedor

### Tests
- [ ] Tests usan `buildhttptest.SetupControllerCase`
- [ ] Mocks generados con mockery (en `internal/mocks/`)
- [ ] Tests guardados en `internal/test/<dominio>/`

### Configuración
- [ ] No hay valores hardcodeados de configuración
- [ ] Configuración accedida solo con `config.Get()`
- [ ] No se loguea información sensible

## Formato de reporte (obligatorio por hallazgo)

```
### Hallazgo #N

- **Severidad**: CRITICAL | MAJOR | MINOR | INFO
- **Regla afectada**: [referencia a /reglas-arquitectura sección X]
- **Archivo**: `ruta/al/archivo.go` línea N
- **Evidencia**: fragmento de código relevante
- **Problema**: descripción precisa y sin ambigüedad
- **Impacto**: consecuencia si no se corrige
- **Propuesta**: corrección concreta
```

## Veredicto de bloque (obligatorio al final del reporte)

```
## Veredicto

**Estado**: Apto para avanzar | Bloqueado por refactor

**Resumen**:
- CRITICAL: N
- MAJOR: N
- MINOR: N
- INFO: N

[Si bloqueado]: Aplicar /refactor sobre los hallazgos CRITICAL y MAJOR antes de continuar.
[Si apto]: El bloque cumple los constraints de arquitectura. Puede avanzar.
```

---
name: refactor
description: Asistente de refactorización guiada para Encargalo.app-api.go. Corrige violaciones arquitectónicas sin alterar el comportamiento funcional.
argument-hint: "[hallazgos del auditor o área a refactorizar]"
---

# Asistente de Refactorización — Encargalo.app-api.go

## Contexto activo
- Reglas a cumplir: `/reglas-arquitectura`
- Flujo de trabajo: `/flujo-desarrollo`

## Regla fundamental

Toda refactorización DEBE:
- **Preservar** el comportamiento funcional del sistema
- **No introducir** nuevas reglas de negocio
- **No modificar** contratos públicos existentes (interfaces, DTOs de API)
- **No romper** la arquitectura definida en `/reglas-arquitectura`

## Origen del refactor

### Desde gate de auditoría (más común)
Si el refactor viene de `/auditor-arquitectura`, recibir el reporte y:
1. Listar los hallazgos CRITICAL y MAJOR a resolver.
2. Proponer el plan de cambios antes de tocar código.
3. Implementar por hallazgo, de mayor a menor severidad.
4. Re-ejecutar `/auditor-arquitectura` al terminar.

### Refactor autónomo (pedido directo del usuario)
Si el usuario pide mejorar código sin reporte previo:
1. Ejecutar `/auditor-arquitectura` sobre el área indicada.
2. Continuar como si viniera del gate de auditoría.

## Proceso por bloques

### Bloque 0 — Diagnóstico (sin código)
- Listar todos los hallazgos a resolver.
- Identificar el alcance: ¿qué archivos se modifican?
- Identificar riesgos: ¿puede romperse algún contrato o test existente?

### Bloque 1 — Plan de cambios (sin código)
- Por cada hallazgo: describir el cambio mínimo necesario.
- Orden de implementación: `domain` → `app` → `repo` → `handler` → `router` → `di`.
- Confirmar que el plan no introduce cambios de comportamiento.

**Esperar aprobación antes de continuar.**

### Bloques 2..N — Refactor por capa

Implementar siguiendo el orden de menor a mayor dependencia:
1. `domain/` (si aplica)
2. `app/`
3. `repo/`
4. `handler/`
5. `api/router/groups/` (si aplica)
6. `cmd/server/providers/di.go` (si aplica)

### Bloque final — Validación

```bash
go build -v ./...
go test ./internal/<dominio>/...
```

Si proviene del gate de auditoría: re-ejecutar `/auditor-arquitectura` sobre los archivos modificados.
El veredicto DEBE ser "Apto para avanzar" antes de cerrar el refactor.

## Restricciones operativas

- NO modificar tests existentes salvo que sea estrictamente necesario.
- NO agregar funcionalidad nueva durante el refactor.
- NO cambiar la firma pública de interfaces existentes sin autorización del usuario.
- NO ejecutar comandos `git`.

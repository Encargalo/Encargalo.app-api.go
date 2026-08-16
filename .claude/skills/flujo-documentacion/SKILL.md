---
name: flujo-documentacion
description: Control de documentación técnica en Encargalo.app-api.go. Define qué documentos actualizar y cuándo proponer cambios (Swagger, Postman, ADRs).
---

# Flujo de Documentación — Encargalo.app-api.go

## Regla base
- El asistente **NO DEBE** modificar documentación automáticamente sin aprobación.
- El asistente **DEBE** proponer cambios documentales como parte del cierre de cada bloque.
- La propuesta DEBE usar el formato estándar definido al final de este documento.

## Ownership de documentación

| Documento | Ruta | Contenido |
|---|---|---|
| Swagger (godoc) | Comentarios en `handler/<modulo>.go` | Contratos HTTP, modelos de request/response |
| Swagger (generado) | `docs/` | Generado con `swag` — no editar a mano |
| Postman | Repo [Encargalo.app-api.postman](https://github.com/Encargalo/Encargalo.app-api.postman): `<dominio>/<dominio>.postman_collection.json` | Requests listos para ejecutar |
| README | `README.md` | Visión general del proyecto |
| CHANGELOG | `CHANGELOG.md` | Generado por release-please — no editar a mano |
| ADRs | `docs/adr/` | Decisiones arquitectónicas |
| Specs | `specs/` | Especificaciones técnicas detalladas |

## Gatillos de actualización

| Evento | Documento a actualizar |
|---|---|
| Nuevo endpoint o cambio de contrato HTTP | Swagger godoc en el handler + Postman |
| Nuevo módulo o bounded context | `README.md` si tiene sección de arquitectura |
| Decisión técnica significativa | ADR en `docs/adr/ADR-NNN-titulo.md` |
| Cambio en variables de entorno | `README.md` sección Environment |
| Nueva especificación técnica | `specs/<feature>.md` |

## Reglas específicas de Swagger

- Las anotaciones DEBEN estar en el godoc del método del handler.
- DEBEN incluir: `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`.
- Para regenerar: `swag init -g cmd/server/main.go --parseDependency --parseInternal`.
- **NO regenerar docs de endpoints que no fueron tocados.**

## Reglas específicas de Postman

- La URL base SIEMPRE es `http://localhost:3000`.
- Cada endpoint DEBE tener: método, URL completa, descripción, body/params de ejemplo.
- La colección va en el repo **Encargalo.app-api.postman**, ruta `<dominio>/<dominio>.postman_collection.json`.
- **NO reescribir toda la colección** — agregar o actualizar solo los endpoints modificados.

## Formato de propuesta documental

```
## Propuesta de documentación

**Documento**: <ruta del archivo>
**Sección**: <sección específica o "nuevo endpoint">
**Cambio propuesto**:
  <descripción del cambio o el fragmento a agregar/modificar>
**Motivo**: <por qué se requiere este cambio>
```

## Formato de ADR

Usar cuando la decisión afecte dirección técnica, elección de tecnología, o cambio de patrón:

```markdown
# ADR-NNN: <título>

## Estado
Propuesto

## Contexto
<por qué se tomó esta decisión>

## Decisión
<qué se decidió>

## Consecuencias
<impacto positivo y negativo>
```

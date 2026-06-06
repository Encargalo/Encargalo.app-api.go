---
name: reglas-arquitectura
description: Reglas prescriptivas DEBE/NO DEBE para la arquitectura hexagonal de Encargalo.app-api.go. Única fuente de verdad de constraints.
---

# Reglas de Arquitectura — Encargalo.app-api.go

> Estas reglas son **constraints obligatorios**, no sugerencias.
> Fuente única: este archivo. Ningún otro skill introduce reglas prescriptivas nuevas.

## 1. Dependencias entre capas

- `handler/` DEBE depender únicamente de `ports.XxxApp` (interface), nunca del struct de `app/`.
- `app/` DEBE depender únicamente de `ports.XxxRepo` (interface), nunca del struct de `repo/`.
- `repo/` DEBE depender de infraestructura compartida (`shared/adapters/postgres`, `shared/adapters/redis`).
- `domain/` NO DEBE importar nada de `handler/`, `app/` ni `repo/`.
- `shared/` NO DEBE importar nada de los módulos de dominio (`internal/<dominio>/`).

## 2. Responsabilidades por capa

- `domain/models/` DEBE contener solo entidades y value objects — sin lógica HTTP ni de persistencia.
- `domain/ports/` DEBE contener únicamente interfaces `XxxApp` y `XxxRepo`.
- `domain/dtos/` DEBE contener structs de request/response — sin métodos de negocio.
- `handler/` DEBE hacer: bind/validate → llamar al app → mapear errores a HTTP.
- `handler/` NO DEBE contener lógica de negocio.
- `app/` DEBE contener toda la lógica de negocio y orquestación.
- `app/` NO DEBE acceder directamente a la base de datos (solo a través de `ports.XxxRepo`).
- `app/` NO DEBE iniciar transacciones directamente.
- `repo/` DEBE implementar `ports.XxxRepo` y SOLO acceder a infraestructura de persistencia.
- `repo/` NO DEBE contener lógica de negocio.

## 3. Errores de dominio

- Los errores DEBEN usar los centinelas de `errcustom/` (`ErrNotFound`, `ErrUnautorized`, `ErrUnexpectedError`, etc.).
- `handler/` DEBE mapear los errores de `errcustom` a `echo.NewHTTPError` con el código HTTP correcto.
- `app/` NO DEBE devolver errores HTTP — solo errores de dominio.

## 4. Persistencia

- DEBE usarse bun ORM para todas las operaciones de base de datos.
- Las migraciones DEBEN usarse goose — archivos en `internal/shared/adapters/postgres/migrations/`.
- Los archivos de migración DEBEN seguir el formato `000NN_descripcion.sql`.
- El cache de Redis DEBE usarse a través del adaptador `shared/adapters/redis/`.
- Las transacciones DEBEN manejarse en `repo/`.

## 5. Capa HTTP (handler)

- Cada módulo DEBE tener un único struct `XxxHandler` en `handler/<modulo>.go`.
- El struct DEBE recibir `ports.XxxApp` por constructor (DI).
- Los handlers DEBEN tener anotaciones Swagger completas: `@Summary`, `@Description`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`.
- Las rutas DEBEN registrarse en `api/router/groups/<modulo>.go`.
- Las rutas protegidas DEBEN usar `AuthMiddleware.Auth` como primer middleware.
- Las rutas con control de rol DEBEN encadenar `Auth.Require(models.RoleXxx)` después de `Auth`.
- La sesión en handlers DEBE obtenerse con: `ctx.Value(models.SessionKey).(models.ActiveSession)`.
- Los errores HTTP DEBEN crearse con `echo.NewHTTPError(code, message)`.

## 6. Inyección de dependencias

- DEBE usarse uber/dig para toda la DI.
- El registro DEBE hacerse en `cmd/server/providers/di.go`.
- NO DEBE hacerse DI manual fuera del contenedor.

## 7. Documentación

- Cada handler creado o modificado DEBE actualizar las anotaciones Swagger en el mismo acto.
- Cada handler creado o modificado DEBE actualizar la colección Postman en `postman/<dominio>/`.
- La URL base de Postman SIEMPRE es `http://localhost:3000`.
- NO regenerar documentación de endpoints que no fueron tocados.

## 8. Tests

- Los tests de handlers DEBEN usar `buildhttptest.SetupControllerCase`.
- Los mocks DEBEN generarse con mockery usando `.mockery.yaml`.
- Los tests DEBEN guardarse en `internal/test/<dominio>/`.
- NO DEBE modificarse un test existente sin autorización explícita del usuario.

## 9. Configuración y secretos

- El logging DEBE usar `charmbracelet/log`.
- La configuración DEBE cargarse exclusivamente con `config.Get()` (gosuite).
- NO DEBE haber valores de configuración hardcodeados en el código.
- NO DEBE loguearse información sensible (contraseñas, tokens, PII).

## Constraint Layer
Las reglas de este documento son el único contrato prescriptivo del proyecto.
Toda decisión de código se valida contra estas reglas.

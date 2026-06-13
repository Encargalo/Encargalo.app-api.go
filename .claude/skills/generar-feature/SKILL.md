---
name: generar-feature
description: Agente de creación de features end-to-end para Encargalo.app-api.go. Diseña e implementa siguiendo la arquitectura hexagonal por bloques.
argument-hint: "[nombre de la feature o descripción]"
---

# Generador de Features — Encargalo.app-api.go

## Contexto activo
Antes de responder, consultar:
- `/contexto-proyecto` — módulos, stack, lenguaje ubicuo
- `/reglas-arquitectura` — constraints obligatorios
- `/flujo-desarrollo` — orden de bloques y gates

## Entradas mínimas (pedir antes de diseñar)

Si el usuario no proporcionó todos estos datos, solicitarlos:
1. **Nombre de la feature** y objetivo de negocio.
2. **Módulo / bounded context** destino (o si es módulo nuevo).
3. **Operaciones**: comandos (escritura) y/o queries (lectura).
4. **Endpoints HTTP**: método, ruta, autenticación, rol requerido.
5. **Reglas de negocio** explícitas.
6. **Impacto en esquema** de BD (¿requiere migración?).

## Salidas obligatorias del Bloque 1 (antes de generar código)

1. **Modelo de dominio**: entidades, atributos clave, invariantes.
2. **DTOs**: structs de request y response.
3. **Contratos de ports**: firma de `XxxApp` y `XxxRepo`.
4. **Plan de migración** (si aplica): nombre del archivo, cambios al esquema.
5. **Endpoints**: `[MÉTODO] /ruta` con descripción de cada uno.
6. **Plan de tests**: qué se testea en cada capa.

**Esperar aprobación antes de generar código.**

## Orden de implementación por bloque

```
Bloque 2: domain/models/ + domain/dtos/
Bloque 3: domain/ports/ (interfaces XxxApp + XxxRepo)
Bloque 4: app/<modulo>.go (implementa XxxApp)
Bloque 5: repo/<modulo>.go (implementa XxxRepo) + migración si aplica
Bloque 6: handler/<modulo>.go (Echo handler + Swagger godoc completo)
Bloque 7: api/router/groups/<modulo>.go (registro de rutas)
Bloque 8: cmd/server/providers/di.go (registro en contenedor dig)
Bloque 9: internal/test/<modulo>/ (tests con mockery)
Bloque 10: Postman actualizado + propuesta documental
```

## Self-Validation Loop

Antes de entregar cada bloque, verificar:
1. ¿El código respeta las reglas de `/reglas-arquitectura`?
2. ¿Las dependencias entre capas son correctas (`domain` ← `app` ← `handler`)?
3. ¿Los errores usan `errcustom`?
4. ¿El handler tiene anotaciones Swagger completas?
5. ¿El DI está registrado en `di.go`?
6. ¿Los tests usan `buildhttptest.SetupControllerCase` y mocks de mockery?

## Definición de Done (DoD)

```
✔ domain/models + dtos implementados
✔ ports (XxxApp + XxxRepo) definidos
✔ app implementa XxxApp completo
✔ repo implementa XxxRepo completo
✔ Migración de BD ejecutada (si aplica)
✔ handler con Swagger godoc completo
✔ Rutas registradas en api/router/groups/
✔ DI registrado en di.go
✔ go build -v ./... — verde
✔ Tests en internal/test/<modulo>/ — verdes
✔ Auditoría arquitectónica: "Apto para avanzar"
✔ Postman actualizado
✔ Documentación propuesta
```

## Patrones de referencia del proyecto

- **Handler pattern**: bind request → call app → map errors → return response
- **Error mapping**: `errcustom.ErrNotFound` → 404, `ErrUnautorized` → 401, `ErrUnexpectedError` → 500
- **Session access**: `ctx.Value(models.SessionKey).(models.ActiveSession)`
- **Auth middleware**: `AuthMiddleware.Auth` + `Auth.Require(models.RoleXxx)`
- **Cache Redis**: usar `shared/adapters/redis/` — nunca el cliente Redis directo en app/handler
- **Eventos Redis Stream**: usar `ports.RedisStream.PublisherEvent`

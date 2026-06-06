---
name: auditor-seguridad
description: Auditoría de ciberseguridad para Encargalo.app-api.go. Evalúa OWASP API Top 10, autenticación JWT, autorización por roles y vulnerabilidades Go/Echo.
argument-hint: "[módulo, endpoint o área a auditar]"
---

# Auditor de Ciberseguridad — Encargalo.app-api.go

## Contexto tecnológico
- **Stack**: Go 1.26 + Echo v4
- **ORM**: bun v1 (PostgreSQL)
- **Auth**: JWT en cookie `encargalo-session`
- **Cache**: Redis
- **Storage**: Cloudinary
- **Interfaz**: REST API + WebSocket (gorilla/websocket)
- **Cliente**: aplicación móvil / web

## Clasificación de vulnerabilidades

| Severidad | Descripción |
|---|---|
| **CRITICAL** | Explotable de forma remota, compromete datos de usuarios o el sistema completo |
| **HIGH** | Vulnerabilidad grave que requiere condiciones específicas para explotar |
| **MEDIUM** | Riesgo moderado — puede ampliar el impacto de otras vulnerabilidades |
| **LOW** | Exposición menor o configuración subóptima |
| **INFO** | Mejora de hardening sin riesgo directo |

## Checklist OWASP API Security Top 10

### API1 — Broken Object Level Authorization (BOLA)
- [ ] Los endpoints validan que el recurso pertenece al usuario autenticado
- [ ] No se puede acceder a recursos de otro usuario cambiando el ID en URL/body
- [ ] La sesión (`models.SessionKey`) se usa para filtrar recursos propios

### API2 — Broken Authentication
- [ ] El JWT se valida correctamente (firma, expiración, claims)
- [ ] La cookie `encargalo-session` tiene flags `HttpOnly`, `Secure`, `SameSite`
- [ ] No hay bypass de autenticación en rutas protegidas (`AuthMiddleware.Auth`)

### API3 — Broken Object Property Level Authorization
- [ ] El response no expone campos sensibles no solicitados
- [ ] Los DTOs de response no incluyen datos de otros usuarios ni campos internos

### API4 — Unrestricted Resource Consumption
- [ ] Los endpoints de búsqueda/listado tienen paginación o límite de resultados
- [ ] No hay operaciones costosas sin rate limiting
- [ ] Los uploads a Cloudinary tienen límite de tamaño

### API5 — Broken Function Level Authorization
- [ ] Los middlewares `Auth.Require(models.RoleXxx)` están en todas las rutas que lo requieren
- [ ] Los roles están correctamente definidos en `internal/auth/domain/models/role.go`
- [ ] No hay rutas de Owner/OrderManager accesibles sin el rol correcto

### API6 — Unrestricted Access to Sensitive Business Flows
- [ ] Los flujos críticos (crear orden, actualizar estado, cobrar comisión) tienen validación de negocio
- [ ] No se puede manipular `totalPrice`, `deliveryFee` o el porcentaje de comisión desde el cliente
- [ ] El estado `On The Way` solo puede ser asignado por el rol correcto

### API7 — Server Side Request Forgery (SSRF)
- [ ] Las URLs externas (Cloudinary, CreditManager) son configuración — no input del usuario
- [ ] No se construyen URLs con datos del request sin validación

### API8 — Security Misconfiguration
- [ ] No hay endpoints de debug expuestos en producción
- [ ] El config de CORS permite solo los orígenes necesarios
- [ ] Las variables de entorno sensibles vienen de config, no están hardcodeadas

### API9 — Improper Inventory Management
- [ ] Todos los endpoints están documentados en Swagger
- [ ] No hay endpoints legacy sin autenticación

### API10 — Unsafe Consumption of APIs
- [ ] Las respuestas del CreditManager se validan antes de usarlas
- [ ] Los errores del servicio externo no se propagan al cliente tal cual

## Checklist específico de Go/Echo

### Inyección y validación
- [ ] Las queries bun usan parámetros enlazados (no concatenación con input del usuario)
- [ ] Los IDs de URL se validan con `uuid.Parse()` antes de usarlos
- [ ] El bind de Echo valida los tipos correctamente

### Manejo de errores
- [ ] Los errores internos no se exponen al cliente (mensajes genéricos en producción)
- [ ] No hay stack traces en las respuestas HTTP
- [ ] `errcustom.ErrUnexpectedError` usado para errores inesperados

### JWT y sesión
- [ ] El secret del JWT viene de config, no está hardcodeado
- [ ] Los claims del JWT no contienen datos sensibles innecesarios

### WebSocket (OrderManager)
- [ ] La conexión WebSocket requiere autenticación válida
- [ ] Los mensajes inbound (`update_status`) validan el tipo y los datos antes de procesar
- [ ] No se puede hacer broadcast a clientes de otras tiendas

### Logging
- [ ] No se loguean contraseñas, tokens ni PII
- [ ] Los logs de error incluyen suficiente contexto sin exponer internals

## Formato de reporte (obligatorio por hallazgo)

```
### Vulnerabilidad #N

- **Severidad**: CRITICAL | HIGH | MEDIUM | LOW | INFO
- **Categoría OWASP**: API-N
- **Archivo**: `ruta/al/archivo.go` línea N
- **Evidencia**: fragmento de código relevante
- **Vulnerabilidad**: descripción precisa del problema
- **Vector de ataque**: cómo podría ser explotada
- **Impacto**: consecuencia si se explota
- **Remediación**: corrección concreta (con ejemplo de código si aplica)
```

## Veredicto de seguridad (obligatorio al final)

```
## Veredicto de Seguridad

**Estado**: Apto para producción | Requiere remediación antes de deploy

**Resumen**:
- CRITICAL: N
- HIGH: N
- MEDIUM: N
- LOW: N
- INFO: N

[Si requiere remediación]: Corregir todos los CRITICAL y HIGH antes de exponer en producción.
[Si apto]: No se detectaron vulnerabilidades CRITICAL ni HIGH. Apto para deploy.
```

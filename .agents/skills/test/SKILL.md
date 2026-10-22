---
name: test
description: Crea tests unitarios para handlers y casos de uso cuando no existen. Se invoca manualmente o automáticamente cada que se crea un nuevo handler (controlador) o un caso de uso (app layer). Siempre genera mocks con mockery y guarda los tests en internal/test/. NO modifica tests existentes.
argument-hint: "[domain] [handler|app]"
---

Crea tests unitarios para el dominio indicado (o el que se acaba de crear) siguiendo estrictamente las convenciones del proyecto.

## 0. Verificar si ya existen tests

Antes de hacer cualquier cosa, verifica si ya existe el archivo de test:

```
internal/test/<domain>/handler/<domain>_test.go
internal/test/<domain>/app/<domain>_test.go
```

**Si el archivo ya existe, no lo toques. Termina aquí.**

Solo procede si el archivo NO existe.

## 1. Leer contexto antes de escribir

Lee:
- El handler o app layer a testear
- Los ports/interfaces en `internal/<domain>/domain/ports/`
- Los DTOs en `internal/<domain>/domain/dtos/`
- El `.mockery.yaml` para verificar si el dominio ya está registrado
- Un test existente como referencia: `internal/test/auth/handler/auth_test.go`

## 2. Mocks con mockery

**Siempre** usa mocks generados por mockery. Nunca escribas mocks a mano.

### Agregar el dominio a .mockery.yaml si no existe

```yaml
Encargalo.app-api.go/internal/<domain>/domain/ports:
  config:
    dir: "internal/mocks/<domain>/domain/ports"
```

Si el handler depende de otro dominio (ej. `employee`), agrega ese también.

### Regenerar mocks

```bash
mockery
```

Los mocks quedan en `internal/mocks/<domain>/domain/ports/mock_<Interface>.go`.

## 3. Ubicación de los tests

```
internal/test/<domain>/handler/<domain>_test.go   ← tests de handler
internal/test/<domain>/app/<domain>_test.go        ← tests de app/caso de uso
```

El package siempre es `package handler_test` o `package app_test`.

## 4. Helpers obligatorios en cada archivo de test

```go
func buildBody(t *testing.T, v any) *bytes.Buffer {
    t.Helper()
    b, err := json.Marshal(v)
    if err != nil { t.Fatal(err) }
    return bytes.NewBuffer(b)
}

func httpCode(err error) int {
    if he, ok := err.(*echo.HTTPError); ok {
        return he.Code
    }
    return 0
}

// Para handlers con sesión:
func withSession(tc buildhttptest.ControllerCase, session authmodels.ActiveSession) buildhttptest.ControllerCase {
    ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
    tc.Req = tc.Req.WithContext(ctx)
    tc.Ctx.SetRequest(tc.Req)
    return tc
}
```

## 5. Patrón de cada test

```go
func TestHandlerMethod_Scenario(t *testing.T) {
    // 1. Crear mocks
    svc := mocksprod.NewProductsApp(t)

    // 2. Setup del contexto HTTP
    tc := buildhttptest.SetupControllerCase("/ruta", http.MethodPost, body)
    tc = withSession(tc, session) // si requiere auth

    // 3. Configurar path params si aplica
    tc.Ctx.SetParamNames("param_id")
    tc.Ctx.SetParamValues(id.String())

    // 4. Expectativas del mock
    svc.On("Metodo", tc.Req.Context(), mock.AnythingOfType("dtos.MiDTO")).Return(nil)

    // 5. Ejecutar y afirmar
    h := handler.NewHandler(svc)
    err := h.MetodoHandler(tc.Ctx)

    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, tc.Res.Code)
}
```

### Usar `mock.AnythingOfType` cuando el handler modifica el DTO antes de llamar al servicio

```go
svc.On("RegisterItem", tc.Req.Context(), mock.AnythingOfType("dtos.RegisterItem")).Return(nil)
```

### Usar valores exactos cuando el handler no modifica los argumentos

```go
svc.On("SearchProductsByShopID", tc.Req.Context(), shopID, pagination.Params{Page: 1, Limit: 10}).Return(expected, nil)
```

## 6. Casos a cubrir por cada método

| Escenario | Qué verificar |
|-----------|--------------|
| Success | `assert.NoError` + código HTTP correcto |
| Sin sesión (si requiere auth) | `http.StatusUnauthorized` |
| Body inválido (JSON malformado) | `http.StatusBadRequest` |
| Validación fallida (campos requeridos faltantes) | `http.StatusBadRequest` |
| Parámetro de ruta/query inválido (UUID mal formado) | `http.StatusBadRequest` |
| Parámetro de query requerido ausente | `http.StatusBadRequest` |
| Dependencia no encontrada (ErrNotFound) | error propagado o `http.StatusUnprocessableEntity` |
| Error inesperado (ErrUnexpectedError) | error propagado |

## 7. Errores del dominio

Usa los sentinel errors de `internal/shared/errcustom`. Los handlers los propagan directamente a Echo, así que en el test verifica igualdad directa:

```go
assert.Equal(t, errcustom.ErrNotFound, err)
```

## 8. Query params en tests GET

```go
tc := buildhttptest.SetupControllerCase("/ruta?shop_id="+shopID.String(), http.MethodGet, nil)
tc.Ctx.QueryParams().Set("shop_id", shopID.String())
```

## 9. Ejecutar y verificar

```bash
go test ./internal/test/<domain>/... -v
```

Si hay errores con `mock.AnythingOfType`, el nombre es solo el del struct sin el package prefix (ej. `"RegisterItem"`, no `"dtos.RegisterItem"`).

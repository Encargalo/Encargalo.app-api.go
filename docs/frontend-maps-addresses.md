# Guía Frontend — Direcciones (Google Maps vía backend)

Contrato para la **app cliente**.  
Auth: cookie `encargalo-session` (rol `Customer`).  
Base URL local: `http://localhost:3000`.

Las llamadas a Google **ya no van desde el móvil**. El backend es el proxy (Places Autocomplete, Place Details, Geocoding inverso). La API key de Google no debe vivir en el bundle.

---

## 1. Qué cambió

Antes reverse geocode y place details devolvían solo el string formateado:

```json
{ "formatted_address": "…", "short_address": "…" }
```

```json
{ "address": "…", "lat": 10.49, "lng": -66.85 }
```

Ahora el backend **no arma la dirección**. Pasa el desglose de Google para que el front la construya:

- `parts` — campos planos listos para usar (ciudad, estado, municipio, calle, etc.)
- `address_components` — array crudo de Google, por si hace falta un type que no esté en `parts`

Los campos anteriores (`formatted_address` / `address` / `lat` / `lng`) **siguen existiendo**. El cambio es aditivo.

---

## 2. Endpoints

Todos `GET`, sesión de `Customer`, mismo rate limit.

| Método | Ruta | Para qué |
|---|---|---|
| `GET` | `/maps/places/autocomplete` | Sugerencias mientras el usuario escribe |
| `GET` | `/maps/places/details` | Dirección + coords + `parts` de un `place_id` |
| `GET` | `/maps/geocode/reverse` | Dirección + `parts` a partir de un pin en el mapa |

Límites:

- 30 req / minuto por IP
- 500 req / día por usuario
- Respuesta `429` si se pasan

---

## 3. Flujo A — buscar una dirección (autocomplete)

```text
Usuario escribe (≥ 3 caracteres)
    ↓
GET /maps/places/autocomplete?query=…&session_token=<uuid>&lat=&lng=
    ↓
Lista de sugerencias (place_id + textos)
    ↓
Usuario elige una
    ↓
GET /maps/places/details?place_id=…&session_token=<mismo uuid>
    ↓
parts + coords → armar string y guardar dirección
```

### `GET /maps/places/autocomplete`

| Query | Requerido | Notas |
|---|---|---|
| `query` | sí | Mínimo **3 caracteres** (el servidor recorta espacios) |
| `session_token` | sí | UUID generado en el cliente. **El mismo** se usa luego en details |
| `lat` | no | Sesgo de cercanía (junto con `lng`) |
| `lng` | no | Sesgo de cercanía (junto con `lat`) |

Si `lat`/`lng` no parsean, se ignoran (no es error). Si ambos vienen y están fuera de rango → `400`.

Response `200` — array:

```json
[
  {
    "place_id": "ChIJ…",
    "main_text": "Avenida Francisco de Miranda",
    "secondary_text": "Chacao, Miranda, Venezuela"
  }
]
```

Autocomplete **no** trae ciudad/estado. Eso llega en details.

### `GET /maps/places/details`

| Query | Requerido | Notas |
|---|---|---|
| `place_id` | sí | El de la sugerencia elegida |
| `session_token` | sí | **El mismo UUID** del autocomplete de esa búsqueda |

Response `200`:

```json
{
  "address": "Av. Francisco de Miranda, Chacao 1060, Miranda, Venezuela",
  "lat": 10.497,
  "lng": -66.853,
  "parts": {
    "street_number": "123",
    "route": "Avenida Francisco de Miranda",
    "neighborhood": "Los Palos Grandes",
    "locality": "Chacao",
    "administrative_area_level_2": "Municipio Chacao",
    "administrative_area_level_1": "Miranda",
    "country": "Venezuela",
    "country_code": "VE",
    "postal_code": "1060"
  },
  "address_components": [
    {
      "long_name": "123",
      "short_name": "123",
      "types": ["street_number"]
    }
  ]
}
```

`address` es el `formatted_address` de Google (referencia / fallback). **No lo uses como única fuente** si vas a mostrar ciudad, estado, municipio por separado.

### `session_token`

1. Al abrir el buscador, generar un UUID (`crypto.randomUUID()`).
2. Mandarlo en **todas** las llamadas de autocomplete de esa sesión de tipeo.
3. Mandar **el mismo** en el details de la sugerencia elegida.
4. Cuando el usuario elige un lugar **o** limpia el buscador y empieza otra búsqueda, generar un UUID nuevo.

Google factura autocomplete + details como una sola sesión si el token coincide. Un token distinto en details = cobro extra.

---

## 4. Flujo B — pin en el mapa (reverse geocode)

```text
Usuario mueve el mapa / suelta el pin
    ↓
debounce ~300ms
    ↓
GET /maps/geocode/reverse?lat=10.497&lng=-66.853
    ↓
parts → armar string y prellenar el formulario
```

### `GET /maps/geocode/reverse`

| Query | Requerido | Notas |
|---|---|---|
| `lat` | sí | Número, rango −90…90 |
| `lng` | sí | Número, rango −180…180 |

Response `200`:

```json
{
  "formatted_address": "Av. Francisco de Miranda, Chacao 1060, Miranda, Venezuela",
  "short_address": "123",
  "parts": { "…mismos campos que details…" },
  "address_components": [ "…mismo array que details…" ],
  "results": [
    {
      "formatted_address": "308 Av. Luis Roche, Caracas 1060, Miranda, Venezuela",
      "place_id": "ChIJ…",
      "types": ["street_address"],
      "lat": 10.4971,
      "lng": -66.8479,
      "location_type": "RANGE_INTERPOLATED",
      "parts": { "street_number": "308", "route": "Avenida Luis Roche", "…": "…" },
      "address_components": [ "…" ]
    }
  ]
}
```

Los campos de primer nivel (`formatted_address`, `short_address`, `parts`, `address_components`) salen de `results[0]`. `results` trae **todos** los candidatos que devolvió Google, en el mismo orden, sin filtrar.

`results[0]` muchas veces **no** trae `street_number` (puede ser un `establishment`, un `premise` con Plus Code o un `route`), mientras que un candidato posterior sí. Sugerencia: elegir el primer candidato cuyos `types` incluyan `street_address` o `premise` y que tenga `parts.street_number`; si ninguno lo tiene, el primero que no sea `plus_code`. Ojo: Google a veces pone basura en `street_number` (`PB`, o el mismo valor que `postal_code`).

`short_address` es el primer componente de Google (a veces el número de calle, a veces un Plus Code). Úsalo como label corto del pin si hace falta; para el formulario usa `parts`.

Debounce al arrastrar el mapa. El backend cachea por coords redondeadas a 5 decimales (~1.1 m), pero igual cuenta contra el rate limit.

---

## 5. Cómo leer `parts`

Los nombres son los **types de Google**, no aliases en español. Cualquier campo puede faltar (`omitempty`): no asumas que vienen todos.

| Campo en `parts` | Qué es (VE) | Ejemplo |
|---|---|---|
| `street_number` | Número | `123` |
| `route` | Calle / avenida | `Avenida Francisco de Miranda` |
| `neighborhood` | Urbanización / barrio | `Los Palos Grandes` |
| `sublocality` | Sub-localidad | — |
| `locality` | **Ciudad** | `Chacao` |
| `administrative_area_level_3` | Parroquia / nivel menor | — |
| `administrative_area_level_2` | **Municipio / provincia** | `Municipio Chacao` |
| `administrative_area_level_1` | **Estado** | `Miranda` |
| `country` | País | `Venezuela` |
| `country_code` | ISO del país (`short_name`) | `VE` |
| `postal_code` | Código postal | `1060` |
| `premise` | Edificio / recinto | — |
| `subpremise` | Apto / interior | — |

Si un campo no viene en `parts`, se puede buscar en `address_components` por `types`.

```ts
function componentByType(
  components: { long_name: string; short_name: string; types: string[] }[],
  type: string,
): string | undefined {
  return components.find((c) => c.types.includes(type))?.long_name;
}
```

---

## 6. Cómo armar la dirección (sugerencia)

El backend no concatena. Una armada razonable para el string que se guarda en `address` al crear la dirección del usuario:

```ts
function buildAddressLine(parts: AddressParts): string {
  const street = [parts.route, parts.street_number].filter(Boolean).join(" ");
  const area = parts.neighborhood || parts.sublocality;
  const city = parts.locality;
  const state = parts.administrative_area_level_1;

  return [street, area, city, state].filter(Boolean).join(", ");
}
```

Ejemplo: `Avenida Francisco de Miranda 123, Los Palos Grandes, Chacao, Miranda`.

Para chips / labels del formulario:

```ts
{
  calle: parts.route,
  numero: parts.street_number,
  urbanizacion: parts.neighborhood ?? parts.sublocality,
  ciudad: parts.locality,
  municipio: parts.administrative_area_level_2,
  estado: parts.administrative_area_level_1,
  pais: parts.country,
  codigoPostal: parts.postal_code,
}
```

Si `parts` viene casi vacío, fallback:

- details → `address`
- reverse geocode → `formatted_address`

---

## 7. Guardar la dirección del usuario

`POST` de direcciones del usuario **no cambió**. Sigue pidiendo un string `address` + `lat`/`long`. El front arma ese string con `parts` (sección 6) y manda las coords de details (o las del pin).

Coords:

- Flujo autocomplete → `lat` / `lng` de **details** (no las del sesgo del autocomplete).
- Flujo pin → las del pin (`lat` / `lng` que mandaste a reverse geocode).

---

## 8. Errores

| HTTP | Cuándo | Qué mostrar |
|---|---|---|
| `400` | query &lt; 3, falta `session_token` / `place_id` / `lat`/`lng`, coords fuera de rango | Validación en UI |
| `401` | sin sesión o no es Customer | Login |
| `404` | Google no tiene resultado para esa coord / `place_id` | “No encontramos una dirección aquí” |
| `429` | rate limit | Esperar / reintentar |
| `500` | fallo de Google o interno | Error genérico |

---

## 9. Checklist de implementación

- [ ] Quitar SDK / HTTP directo a Google Places y Geocoding del app (y la API key del bundle).
- [ ] Autocomplete: mínimo 3 caracteres, debounce, `session_token` UUID por sesión de búsqueda.
- [ ] Al elegir sugerencia: details con el **mismo** `session_token`, luego UUID nuevo.
- [ ] Pin en el mapa: reverse geocode con debounce; no disparar en cada frame del drag.
- [ ] Armar el string de dirección desde `parts`, no solo desde `formatted_address` / `address`.
- [ ] Mostrar ciudad (`locality`), estado (`administrative_area_level_1`) y municipio (`administrative_area_level_2`) por separado si el formulario lo pide.
- [ ] Manejar campos ausentes (`parts.locality` puede no venir).
- [ ] Fallback a `address` / `formatted_address` si `parts` está vacío.
- [ ] Guardar coords de details (búsqueda) o del pin (mapa).
- [ ] Manejar `404` y `429`.

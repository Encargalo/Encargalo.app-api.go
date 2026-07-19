# Guía Staff — Onboarding de repartidores (CRM)

Auth: cookie `encargalo-session` + rol `Staff`.  
Base: `http://localhost:3000`.

## Flujo

1. El usuario se registra en la **app cliente** (cuenta Customer).
2. Escribe por WhatsApp; el asesor pide fotos/datos.
3. En CRM → **Repartidores**: buscar por teléfono → guardar borrador (puede ser parcial) → cuando el checklist esté completo → **Activar**.
4. Misma cuenta/contraseña sirve para `POST /auth/sign-in/riders`.

## Checklist para activar

- Foto perfil, identidad, carnet, licencia, carnet de circulación
- 4 fotos del vehículo
- `vehicle_type`
- `vehicle_plate` (no requerida si `vehicle_type = bici`)

## Endpoints

| Método | Ruta | Uso |
|---|---|---|
| GET | `/riders/staff/candidates?phone=` | Buscar cuenta + expediente |
| GET | `/riders/staff/profiles?status=draft\|active\|suspended` | Cola (default `draft`) |
| GET | `/riders/staff/profiles/:userId` | Detalle |
| PUT | `/riders/staff/profiles/:userId` | Upsert parcial (fotos base64/data-URL) |
| DELETE | `/riders/staff/profiles/:userId/photos/:field` | Borra foto (DB + Cloudinary). `field`: `profile_photo`, `id_photo`, `carnet_photo`, `license_photo`, `circulation_card`, `vehicle_photo_1`…`4` |
| POST | `/riders/staff/profiles/:userId/activate` | Rol Rider + `active` |
| POST | `/riders/staff/profiles/:userId/suspend` | Suspende login rider |
| POST | `/riders/staff/profiles/:userId/touch-contact` | Marca follow-up |

Cada respuesta de perfil incluye `completion: { is_complete, missing, filled_count, required_count }`.

### PUT body (parcial)

```json
{
  "vehicle_type": "moto",
  "vehicle_plate": "AB123CD",
  "notes": "Falta licencia, recontactar viernes",
  "id_photo": "data:image/jpeg;base64,..."
}
```

Campos omitidos no se borran. Activar con expediente incompleto → `422` + `missing[]`.

## App cliente (CTA)

Link WhatsApp sugerido (solo front):

`https://wa.me/<NUMERO>?text=Hola,%20quiero%20ser%20repartidor%20Encargalo`

## Fuera de alcance

Snapshot del rider en la orden al aceptar (fase 2).

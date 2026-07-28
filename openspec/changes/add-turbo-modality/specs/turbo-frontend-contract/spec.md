## ADDED Requirements

### Requirement: Contrato UX cliente estilo Rappi
La app cliente SHALL tratar Turbo como vertical de home (tile), no como un ítem más del listado obligatorio de restaurantes. Tras `GET /turbo/entry` con `available=true`, SHALL cargar el catálogo del `shop.id` asignado y mostrar `shop.name` en carrito y checkout.

#### Scenario: Tile solo si hay cobertura
- **WHEN** entry retorna `available=false`
- **THEN** el front oculta el tile Turbo o muestra estado no disponible en la zona

#### Scenario: Carrito muestra el nodo
- **WHEN** el usuario tiene ítems turbo en el carrito
- **THEN** el header/resumen muestra el `name` del shop asignado (ej. “Turbo Fresh”), no solo la palabra genérica “Turbo”

#### Scenario: Cambio de dirección
- **WHEN** el usuario cambia la dirección de entrega y el nodo previo queda inválido
- **THEN** el front vuelve a llamar entry y vacía o invalida el carrito turbo si el `shop.id` cambia o no hay cobertura

---

### Requirement: Documentación frontend entregable
El cambio SHALL incluir un documento `docs/frontend-turbo.md` con el flujo tile → entry → products → order, códigos de error y checklist de UI.

#### Scenario: Doc presente
- **WHEN** se completa el change
- **THEN** existe `docs/frontend-turbo.md` usable por el equipo de app cliente

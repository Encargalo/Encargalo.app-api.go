## ADDED Requirements

### Requirement: Búsqueda de cuenta por teléfono para vinculación
Un usuario con rol `Staff` SHALL poder buscar una cuenta existente por número de teléfono para vincularla como `Owner` de un shop. La búsqueda SHALL devolver los datos identificatorios de la cuenta (nombre, teléfono) para confirmación antes de asignar el rol.

#### Scenario: Staff busca una cuenta existente
- **WHEN** el staff busca por un teléfono que corresponde a una cuenta ya registrada
- **THEN** el sistema devuelve el nombre y teléfono de esa cuenta

#### Scenario: Staff busca un teléfono sin cuenta asociada
- **WHEN** el staff busca por un teléfono que no corresponde a ninguna cuenta registrada
- **THEN** el sistema responde que no se encontró ninguna cuenta, sin crear una nueva

### Requirement: Asignación del rol Owner sin remover roles existentes
Un usuario con rol `Staff` SHALL poder asignar el rol `Owner` (asociado a un `shop_id` específico) a una cuenta ya existente. Esta asignación SHALL agregarse a los roles que la cuenta ya tenga, sin removerlos.

#### Scenario: Cuenta Customer se vincula como Owner
- **WHEN** el staff asigna el rol Owner de un shop a una cuenta que hoy solo tiene el rol Customer
- **THEN** la cuenta queda con ambos roles, Customer y Owner, y puede iniciar sesión de negocio con el rol Owner sobre ese shop

#### Scenario: Cuenta ya vinculada a otro shop como Owner
- **WHEN** el staff intenta asignar el rol Owner de un shop distinto a una cuenta que ya es Owner de otro shop
- **THEN** el sistema aplica la misma regla de asignación (agrega el rol adicional), sin remover la vinculación previa

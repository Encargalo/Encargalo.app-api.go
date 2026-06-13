# GUÍA: Replicar el Sistema AI Multi-Agent para tu Proyecto

Este documento explica paso a paso cómo construir tu propio sistema de prompts para controlar el comportamiento de Claude Code (o cualquier LLM) como arquitecto principal de cualquier tipo de proyecto de software.

---

## ¿Qué es este sistema?

Es un **AI Multi-Agent Development System** basado en archivos Markdown. No usa frameworks externos. Funciona porque Claude Code carga el contexto de los archivos como instrucciones del sistema.

**El principio central:** un archivo actúa como *router* (el System Architect) y delega a agentes especializados según la intención del usuario. La IA nunca explica el routing — simplemente responde como el agente correcto.

### Beneficios
- El asistente siempre conoce la arquitectura, convenciones y reglas de tu proyecto.
- Las reglas son versionables, revisables y evolucionan con el código.
- Los errores recurrentes se corrigen una vez en el prompt, no en cada conversación.
- Cada tipo de tarea (crear feature, auditar, refactorizar, documentar) tiene su propio checklist y formato de salida.

---

## Mapa del sistema (11 archivos)

```
docs/prompts/
├── 0_system_architect.md       ← ROUTER principal + contrato operativo
├── 1_project_context.md        ← Mapa del dominio, módulos, lenguaje ubicuo
├── 2_architecture_rules.md     ← Reglas prescriptivas DEBE/NO DEBE (única fuente)
├── 3_development_workflow.md   ← Flujo por bloques, gates, orden de capas
├── 4_documentation_workflow.md ← Cuándo y cómo actualizar /docs
├── 5_feature_generator.md      ← Agente: crear features end-to-end
├── 6_architecture_auditor.md   ← Agente: auditoría arquitectónica
├── 7_refactor_assistant.md     ← Agente: refactorización guiada
├── 8_prompt_evolution.md       ← Agente: evolución del sistema de prompts
├── 9_user_story_generator.md   ← Agente: generación de Historias de Usuario
└── 10_ciber_seguridad_auditor.md ← Agente: auditoría de ciberseguridad
```

### Jerarquía de ownership (anti-duplicación)

| Archivo | Dueño de |
|---|---|
| `2_architecture_rules.md` | Reglas prescriptivas DEBE/NO DEBE |
| `0_system_architect.md` | Modos, routing, contrato de interacción |
| `1_project_context.md` | Mapa de dominio, módulos, lenguaje ubicuo |
| `3_development_workflow.md` | Proceso por bloques y gates |
| `4_documentation_workflow.md` | Qué documentos se actualizan y cómo |
| `5–10_*.md` | Plantillas operativas especializadas |

---

## Relación con Spec Driven Development (SDD)

Este sistema aplica los principios del **Spec Driven Development**: las especificaciones se definen primero y el código se valida contra ellas.

En SDD clásico: definir spec → implementar → validar que la implementación cumple la spec.

En este sistema:

| Concepto SDD | Equivalente aquí |
|---|---|
| Especificación | `2_architecture_rules.md` + `1_project_context.md` |
| Implementación guiada | `5_feature_generator.md` |
| Validación contra spec | `6_architecture_auditor.md` |
| Contrato de proceso | `3_development_workflow.md` |
| Spec evolutiva | `8_prompt_evolution.md` |

**La diferencia respecto a SDD tradicional:** en SDD clásico las specs son artefactos de código (OpenAPI, JSON Schema, tipos). Aquí son texto estructurado en Markdown, legibles por humanos y por LLMs. El asistente las interpreta como restricciones operativas, no como tests automatizados.

Esto es útil cuando los contratos aún no están formalizados, cuando el equipo tiene convenciones implícitas que quiere codificar, o cuando se necesita coherencia de arquitectura a lo largo del tiempo sin un toolchain de validación estática.

---

## ¿Desde cero o proyecto existente?

Antes de crear los archivos, define tu punto de partida. El sistema funciona en ambos casos, pero el proceso de llenado difiere.

| Dimensión | Greenfield (desde 0) | Brownfield (ya existe código) |
|---|---|---|
| Punto de partida | Proyecto nuevo, sin código | Código ya existe en producción o desarrollo |
| `1_project_context.md` | Define lo que vas a construir | Documenta lo que ya existe |
| `2_architecture_rules.md` | Reglas ideales desde el inicio | Reglas reales actuales + aspiracionales separadas |
| Paso 0 | Omitir | **Obligatorio** |
| Agentes | Diseñan sin restricciones heredadas | Respetan patrones y convenciones existentes |

> **Brownfield**: Ejecuta siempre el **Paso 0** antes de crear los prompts. Los archivos `1_project_context.md` y `2_architecture_rules.md` deben reflejar el estado **actual** del código, no el ideal.

---

## Paso a paso para construir tu sistema

### Paso 0 — Diagnóstico del repositorio (solo Brownfield)

Si ya existe código, ejecuta este paso antes de crear cualquier prompt. Pídele al asistente que analice el estado actual:

```
Analiza este repositorio y genera un reporte con:
1. Estructura de carpetas y módulos principales.
2. Stack tecnológico: lenguaje(s), frameworks, versiones detectadas.
3. Patrón arquitectónico aparente (MVC, Clean, Hexagonal, MVVM, etc.).
4. Convenciones de naming detectadas (sufijos, prefijos, idioma del código).
5. Patrones recurrentes en archivos del mismo tipo.
6. Inconsistencias o anti-patrones visibles.
7. Grupos de funcionalidad identificados (módulos, bounded contexts, dominios).

No generes código. Solo analiza y reporta.
```

Usa ese reporte para completar los Pasos 4 y 5 con datos reales, no con suposiciones.

En `2_architecture_rules.md`, distingue explícitamente:
- **Reglas aplicadas hoy**: lo que el código ya hace de forma consistente.
- **Reglas aspiracionales**: lo que debería hacer pero aún no cumple (deuda técnica).

---

### Paso 1 — Elegir la ruta y crear la carpeta de prompts

Antes de crear la carpeta, decide dónde vivirán los prompts. No hay una ruta obligatoria.

**Sugerencias:**

| Ruta | Cuándo usarla |
|---|---|
| `docs/prompts/` | El proyecto ya tiene una carpeta `docs/` establecida |
| `.claude/prompts/` | Los prompts son configuración del asistente, no documentación del proyecto |
| `ai/` | Se quiere separar herramientas de IA del código y la documentación |
| `prompts/` | Proyectos pequeños o sin estructura de `docs/` |
| `[módulo]/prompts/` | Monorepos donde cada módulo tiene su propio contexto |

Una vez elegida, crea la carpeta:

```bash
mkdir [TU-RUTA]
```

> A lo largo de esta guía, `docs/prompts/` es la ruta de ejemplo. Reemplaza todas las referencias con la ruta que elegiste y regístrala en la variable **Ruta de prompts** en la tabla de personalización.

**¿Agregar al `.gitignore`?**

Por defecto, los prompts deben estar versionados — son parte del contrato de desarrollo del equipo. Sin embargo, puedes excluirlos si:
- Contienen información sensible del negocio que no debe exponerse en el repositorio.
- El proyecto es open source y los prompts son parte de un sistema privado.

```gitignore
# Solo si decides no versionar los prompts
[TU-RUTA]/
```

---

### Paso 2 — Crear `CLAUDE.md` (punto de entrada)

Este archivo es lo primero que Claude Code lee al abrirse en el proyecto. Debe contener:

1. Comandos del proyecto (build, test, run, publish).
2. Resumen de arquitectura (capas, estructura de carpetas).
3. Convenciones de naming y estilo de código.
4. Referencias a los documentos de `/docs`.

**Plantilla base:**

```markdown
# CLAUDE.md

## Commands
\`\`\`bash
# Build
[tu-comando-build]

# Run
[tu-comando-run]

# Tests
[tu-comando-test]
\`\`\`

## Architecture
[Descripción breve de capas y proyectos]

## Conventions
### Language
- Código técnico: inglés
- Dominio/negocio: [español u otro lenguaje ubicuo]

### Naming
[tus convenciones de naming]

## Documentation
- `docs/architecture.md` — visión de capas
- `docs/conventions.md` — convenciones completas
- `docs/adr/` — Architecture Decision Records
```

**Cómo enlazar el sistema de prompts desde CLAUDE.md:**

Al final de tu CLAUDE.md, puedes agregar una nota que invite a Claude a cargar el contexto:

```markdown
## AI Development System
Para desarrollo asistido, el asistente debe cargar el contexto de:
`docs/prompts/0_system_architect.md`
```

> **Nota**: Claude Code lee CLAUDE.md automáticamente. Los archivos de `docs/prompts/` se cargan cuando el usuario los menciona o cuando el asistente los referencia en su contexto.

---

### Paso 3 — Crear `0_system_architect.md` (el router)

Este es el archivo más importante. Define:
- El **rol** del asistente (arquitecto principal).
- Las **reglas de routing** (qué agente se activa para cada tipo de tarea).
- Los **modos de trabajo** (FEATURE, AUDITORÍA, REFACTOR, etc.).
- La **política de bloques** (no cerrar sin auditoría).

**Variables que debes personalizar:**

```markdown
# SYSTEM ARCHITECT — CONTRATO OPERATIVO ([TU-PROYECTO])

## Rol
- Actúas como Arquitecto de Software Principal para [descripción de tu sistema].
- Priorizas: arquitectura correcta > mantenibilidad > claridad del dominio > consistencia.

## Regla fundamental (contexto obligatorio)
- Antes de responder cualquier solicitud, DEBES basarte en el contexto definido en:
  - `docs/prompts/1_project_context.md`
  - `docs/prompts/2_architecture_rules.md`
  - `docs/prompts/3_development_workflow.md`
  - `docs/prompts/4_documentation_workflow.md`
```

**Sección de routing (personaliza los triggers):**

```markdown
## Agent Routing

Creación o diseño de nuevas features
→ Delegar a: `5_feature_generator.md`

Auditoría de arquitectura o revisión de código
→ Delegar a: `6_architecture_auditor.md`

Refactorización o mejora de código existente
→ Delegar a: `7_refactor_assistant.md`

Documentación o cierre de feature
→ Delegar a: `4_documentation_workflow.md`

Auditoría de ciberseguridad | Security assessment | OWASP
→ Delegar a: `10_ciber_seguridad_auditor.md`

Evolución del sistema de prompts
→ Delegar a: `8_prompt_evolution.md`

Crear Historia de Usuario | Crear HU | Backlog
→ Delegar a: `9_user_story_generator.md`
```

**Reglas clave que NO debes omitir:**

```markdown
## Router Response Behavior
- El System Architect no debe explicar el proceso de routing al usuario.
- El routing debe ser interno y silencioso.
- Solo debe responder con la salida producida por el agente delegado.

## Política estándar: no cerrar un bloque sin auditoría
- Ningún bloque se declara "listo" sin ejecutar auditoría arquitectónica.
- Si hay hallazgos CRITICAL o MAJOR, se aplica refactor y luego se re-audita.
- Solo cuando la re-auditoría confirma "apto para avanzar", se solicita aprobación.

## Idioma y comunicación
- Las respuestas DEBEN estar en [tu idioma].
- Mensajes de commit: [tus reglas de commit].
```

---

### Paso 4 — Crear `1_project_context.md` (el mapa del dominio)

Este archivo define **quién eres** como proyecto. El asistente lo usa para mantenerse coherente con el lenguaje ubicuo, los módulos y los patrones existentes.

**Variables clave a definir:**

```markdown
# PROJECT CONTEXT — MAPA OPERATIVO ([TU-PROYECTO])

## Identidad del sistema
- Nombre: [NOMBRE-DEL-SISTEMA]
- Tipo: [API REST / SPA frontend / aplicación móvil / CLI / biblioteca / fullstack / microservicio / otro]
- Target: [lenguaje y versión: .NET 10, Node 22, Java 21, etc.]
- Arquitectura: [Clean Architecture + DDD / Hexagonal / MVC / etc.]

## Coordenadas del repositorio
- Carpeta de documentación: `/docs`
- Proyectos/módulos:
  - [Proyecto 1] — rol
  - [Proyecto 2] — rol

## Naming Conventions
- Servicios: [tus convenciones]
- DTOs: [tus convenciones]
- Repositorios: [tus convenciones]

## Localización y lenguaje ubicuo
- Dominio en [español/inglés/etc.]
- Glosario:
  - [Término 1]: [definición]
  - [Término 2]: [definición]

## Contextos delimitados (bounded contexts / módulos)
### [NombreModulo1]
- Propósito: [qué hace]
- En código: [ruta]
- Entidades principales: [lista]

### [NombreModulo2]
...

## Patrones y convenciones ya presentes
- [Patrón 1]: [descripción breve]
- [Patrón 2]: [descripción breve]

## Deuda técnica conocida (solo Brownfield)
- [Área o módulo]: [descripción del problema y por qué aún existe]
```

> **Brownfield**: Esta sección la derivas del Paso 0. No inventes patrones — documenta lo que el código realmente hace. Si hay inconsistencias entre módulos, descríbelas aquí.

**Qué incluir en el glosario:**
- Términos del negocio que no son obvios (estados, transiciones, conceptos del dominio).
- Abreviaciones usadas en el código.
- Sinónimos que podrían confundir al asistente.

---

### Paso 5 — Crear `2_architecture_rules.md` (las reglas prescriptivas)

Este es el archivo más crítico. Es la **única fuente de verdad** para reglas del tipo "DEBE / NO DEBE". Ningún otro archivo debe introducir reglas prescriptivas nuevas.

**Estructura recomendada:**

```markdown
# ARCHITECTURE RULES — CONTRATO PRESCRIPTIVO ([TU-PROYECTO])

## 1) Reglas de capas
### 1.1 Dependencias permitidas
- [Capa A] DEBE depender de [Capa B].
- [Capa C] NO DEBE depender de [Capa D].

### 1.2 Reglas de ubicación de responsabilidades
- [Capa X] DEBE contener [responsabilidad].
- [Capa Y] NO DEBE contener [responsabilidad].

## 2) Reglas de dominio (si aplica DDD)
### 2.1 Entidades y agregados
### 2.2 Dependencias del dominio
### 2.3 Excepciones de dominio

## 3) Reglas de aplicación (casos de uso)
### 3.1 Servicios por módulo
### 3.2 Persistencia
### 3.3 Manejo de errores (Result Pattern o equivalente)
### 3.4 Validación
### 3.5 Mapeo

## 4) Reglas de persistencia/infraestructura
### 4.1 ORM o acceso a datos
### 4.2 Transacciones
### 4.3 Consultas

## 5) Reglas de interfaz / capa de salida (adaptar según el tipo de proyecto)
Personaliza esta sección según lo que aplique a tu stack. Ejemplos:
- **API REST**: controllers, contrato de respuesta (estructura JSON), autorización.
- **Frontend SPA**: componentes de página, props permitidas, manejo de estado global, routing.
- **CLI**: comandos disponibles, flags, formato de output y mensajes de error.
- **Biblioteca**: API pública exportada, contratos de función, convenciones de error.
- **Móvil**: pantallas, navegación, permisos del sistema.
### 5.1 [Interfaz principal según el tipo de proyecto]
### 5.2 Contrato de salida
### 5.3 Acceso y seguridad (si aplica)

## 6) Logging
## 7) Configuración y secretos

## Constraint Layer
Las reglas definidas en este documento constituyen la capa de restricciones.
Estas reglas no son sugerencias. Son constraints obligatorios.
```

**Cómo definir tus reglas:**
- Usa siempre DEBE o NO DEBE (nunca "debería" o "podría").
- Cada regla es una oración corta y verificable.
- Si no estás seguro si algo es una regla, ponlo en `1_project_context.md` como convención.

---

### Paso 6 — Crear `3_development_workflow.md` (el flujo por bloques)

Define **cómo se trabaja**, no qué se construye. Incluye el orden de capas, los gates de verificación, y cuándo se puede avanzar.

**Estructura esencial:**

```markdown
# DEVELOPMENT WORKFLOW — EJECUCIÓN POR BLOQUES ([TU-PROYECTO])

## Modelo de trabajo por bloques
- Un "bloque" es una entrega pequeña, revisable y reversible.
- En modo implementación, se produce código solo para el bloque actual.

## Orden obligatorio por capa
- 1) [Capa base / Dominio]
- 2) [Casos de uso / Aplicación]
- 3) [Persistencia / Infraestructura]
- 4) [API / Presentación]
- 5) Tests
- 6) Cierre (validación + documentación)

## Bloque 0 — Contexto y alcance (sin código)
## Bloque 1 — Diseño (sin código)
## Bloque 2..N — Implementación por capa

## Gates de verificación (antes de cerrar un bloque)
- Verificar consistencia con `2_architecture_rules.md`.
- Verificar que los tests del alcance modificado estén verdes.

## Gate obligatorio: auditoría + refactor por bloque
1) Ejecutar verificación (build/tests).
2) Ejecutar auditoría con `6_architecture_auditor.md`.
3) Si hay CRITICAL/MAJOR: refactor + re-auditoría.
4) Solo entonces: declarar bloque como "listo".

## Regla operativa: el asistente NO ejecuta operaciones git
- El asistente NUNCA debe ejecutar comandos git sin instrucción explícita del usuario.

## Cierre
- Proponer actualizaciones a `/docs` según `4_documentation_workflow.md`.
```

**Adaptar el orden de capas:**
Si tu arquitectura no es Clean Architecture, ajusta los bloques. Por ejemplo:
- Para **MVC clásico**: Model → Controller → View → Tests.
- Para **Hexagonal**: Domain → Ports → Adapters → Tests.
- Para **microservicios**: Contract → Service → Infrastructure → API → Tests.

---

### Paso 7 — Crear `4_documentation_workflow.md` (control documental)

Define cuándo y cómo se actualiza `/docs`. La regla base: el asistente **propone** cambios, nunca los aplica sin autorización.

```markdown
# DOCUMENTATION WORKFLOW ([TU-PROYECTO])

## Regla base
- El asistente NO DEBE modificar documentación automáticamente.
- El asistente DEBE proponer cambios como parte del cierre de una entrega.

## Ownership de documentación (qué va en cada archivo)
- `docs/architecture.md` — visión general por capas
- `docs/domain.md` — entidades, agregados, invariantes
- `docs/api.md` — endpoints, contratos, autenticación
- `docs/changelog.md` — registro de cambios
- `docs/decisions/ADR-*` — decisiones arquitectónicas

## Gatillos de actualización (cuándo proponer cambios)
- Si cambia el contrato HTTP → proponer cambios en `api.md`.
- Si se añaden entidades/invariantes → proponer cambios en `domain.md`.
- Si cambia el modelo de capas → proponer cambios en `architecture.md`.
- Si la decisión afecta dirección técnica → proponer ADR.

## Formato de propuesta (salida obligatoria)
- Archivo:
- Sección:
- Cambio propuesto:
- Motivo:
```

---

### Paso 8 — Crear `5_feature_generator.md` (agente de features)

Este agente se activa cuando el usuario pide crear o implementar funcionalidad. Su trabajo es producir la feature end-to-end siguiendo el orden de capas.

**Secciones clave:**

```markdown
# FEATURE GENERATOR ([TU-PROYECTO])

## Referencias activas (cargar primero)
- `docs/prompts/1_project_context.md`
- `docs/prompts/2_architecture_rules.md`
- `docs/prompts/3_development_workflow.md`
- `docs/prompts/4_documentation_workflow.md`

## Entradas mínimas (solicitar antes de proponer código)
- Nombre de la feature y objetivo de negocio.
- Módulo / bounded context destino.
- Operaciones esperadas (comandos y/o consultas).
- Interfaz de salida esperada (endpoint, componente, comando, función pública, evento, etc.).
- Reglas de negocio explícitas.

## Salidas obligatorias (antes de código)
- Modelo de dominio (entidades, invariantes, métodos).
- Diseño por capas (qué cambia en cada capa).
- Plan de pruebas por capa.

## Self-Validation Loop
Antes de entregar, el agente debe verificar:
1. ¿El código respeta las reglas de arquitectura?
2. ¿Las dependencias entre capas son correctas?
3. ¿El diseño sigue el workflow de `3_development_workflow.md`?

## Definition of Done (DoD)
✔ Capa de dominio implementada
✔ Casos de uso definidos
✔ Persistencia configurada
✔ API creada (si aplica)
✔ Tests en verde
✔ Auditoría arquitectónica aprobada
✔ Documentación propuesta
```

---

### Paso 9 — Crear `6_architecture_auditor.md` (agente de auditoría)

Este agente revisa el código y detecta violaciones de arquitectura. Es el gate de calidad del sistema.

**Secciones clave:**

```markdown
# ARCHITECTURE AUDITOR ([TU-PROYECTO])

## Clasificación de problemas
### CRITICAL — Violaciones graves que comprometen mantenibilidad
### MAJOR — Problemas de diseño que generan deuda técnica
### MINOR — Problemas de calidad o consistencia

## Checklist de auditoría
### [Capa 1 / Dominio]
- [Ítem de verificación 1]
- [Ítem de verificación 2]

### [Capa 2 / Aplicación]
...

## Uso como gate de bloque
- Emitir veredicto explícito:
  - "Apto para avanzar" si no existen CRITICAL/MAJOR pendientes.
  - "Bloqueado por refactor" si existe al menos un CRITICAL/MAJOR.

## Formato de salida (obligatorio)
Para cada hallazgo:
- Severidad: CRITICAL / MAJOR / MINOR
- Regla afectada: referencia a `2_architecture_rules.md`
- Evidencia: archivo(s) y fragmento relevante
- Problema: descripción sin ambigüedad
- Impacto
- Propuesta de refactor
```

---

### Paso 10 — Crear `7_refactor_assistant.md` (agente de refactor)

Se activa después de una auditoría con hallazgos, o cuando el usuario pide mejorar código existente.

**Principio fundamental:**

```markdown
## Regla fundamental de refactorización
Toda refactorización debe:
- preservar el comportamiento funcional del sistema
- no introducir nuevas reglas de negocio
- no modificar contratos públicos existentes
- no romper la arquitectura definida
```

**Proceso esencial:**

```markdown
## Proceso de refactor (orden y bloques)
### Bloque 0 — Diagnóstico (sin código)
### Bloque 1 — Plan de cambios (sin código)
### Bloque 2..N — Refactor por capa
### Bloque final — Tests y validación
- Si el refactor proviene del gate de auditoría, re-ejecutar `6_architecture_auditor.md`.
```

---

### Paso 11 — Crear `8_prompt_evolution.md` (gobernanza del sistema)

Este agente controla cómo evoluciona el sistema de prompts. Es el más "meta" del sistema.

**Reglas clave de gobernanza:**

```markdown
## Ownership de reglas (anti-duplicación)
- Las reglas prescriptivas (DEBE/NO DEBE) viven EXCLUSIVAMENTE en `2_architecture_rules.md`.
- Ningún otro prompt puede introducir nuevas reglas prescriptivas.
- Los demás prompts referencian reglas existentes, no las duplican.

## Cuándo evolucionar prompts
- Cuando un hallazgo se repite en auditorías o refactors.
- Cuando se agregan convenciones nuevas al proyecto.
- Cuando cambia el stack o herramientas.

## Cuándo NO evolucionar prompts
- Para documentar una excepción puntual.
- Para duplicar reglas ya existentes.

## Proceso de cambio
1. Identificar el problema y su evidencia.
2. Determinar ownership (¿es regla prescriptiva, workflow, o contexto?).
3. Aplicar cambio mínimo en el archivo correcto.
4. Validar que no hay duplicación ni contradicción.
```

---

### Paso 12 — Crear `9_user_story_generator.md` (agente de HU)

Opcional pero muy útil. Genera Historias de Usuario estructuradas listas para el backlog.

```markdown
# USER STORY GENERATOR ([TU-PROYECTO])

## Propósito
Generar Historias de Usuario completas, implementables y alineadas con la arquitectura del proyecto.

## Contexto obligatorio
- `docs/prompts/1_project_context.md`
- `docs/prompts/2_architecture_rules.md`
- `docs/prompts/3_development_workflow.md`
- [Si tienes un marco estandarizado: `docs/user_stories/marco_estandarizado_HU.md`]

## Reglas
- No inventar estructura nueva.
- Si el requerimiento es ambiguo, pedir aclaración.

## Output
- La HU debe seguir el marco definido.
- No generar código.
- El resultado será consumido por `5_feature_generator.md`.
```

---

### Paso 13 — Crear `10_ciber_seguridad_auditor.md` (agente de seguridad)

Este agente realiza auditorías de ciberseguridad (SAST/DAST/API). Puedes copiar el archivo existente y adaptarlo a tu stack tecnológico.

**Variables a personalizar:**

```markdown
# Contexto tecnológico del proyecto
- Stack: [tu stack]
- ORM: [tu ORM]
- Auth: [tu mecanismo de autenticación / N/A]
- Interfaz: [REST / GraphQL / gRPC / WebSocket / CLI / SDK / componentes UI / otro]
- Tipo de cliente: [web / móvil / servidor / herramienta / otro]
```

---

## Cómo activar el sistema en una nueva sesión

### Opción A: Cargar manualmente

Al inicio de tu conversación con Claude Code, di:

```
Actúa como el System Architect de este proyecto.
Carga el contexto desde: docs/prompts/0_system_architect.md
```

### Opción B: Referenciar en CLAUDE.md

Agrega al final de tu `CLAUDE.md`:

```markdown
## Sistema de Asistencia IA
Para trabajar con asistencia IA, carga el contexto desde:
`docs/prompts/0_system_architect.md`

Este archivo define el comportamiento del asistente como arquitecto principal,
incluyendo reglas de arquitectura, flujo de trabajo por bloques y agentes especializados.
```

### Opción C: Incluir en el prompt inicial (más directo)

```
Lee el archivo docs/prompts/0_system_architect.md y actúa según ese contrato.
Quiero [describir tu tarea].
```

---

## Variables de personalización por archivo

| Variable | Archivo | Qué personalizar |
|---|---|---|
| Modo del proyecto | `0_`, `1_`, `2_` | Greenfield (desde 0) o Brownfield (ya existe código) |
| Tipo de proyecto | Todos los agentes | API backend / SPA frontend / CLI / biblioteca / móvil / fullstack |
| Ruta de prompts | Todos los archivos | La carpeta elegida en Paso 1 (reemplaza `docs/prompts/` en toda la guía) |
| Nombre del proyecto | `0_`, `1_`, `2_` | Reemplazar `[TU-PROYECTO]` |
| Stack tecnológico | `1_project_context.md` | Lenguaje, framework, versión |
| Arquitectura | `2_architecture_rules.md` | Capas, dependencias permitidas |
| Módulos/bounded contexts | `1_project_context.md` | Lista de módulos y responsabilidades |
| Glosario | `1_project_context.md` | Términos del negocio |
| Idioma de respuestas | `0_system_architect.md` | Español / Inglés / otro |
| Reglas de commit | `0_system_architect.md` | Formato de mensajes git |
| Orden de capas | `3_development_workflow.md` | Bloques de implementación |
| Docs de referencia | `4_documentation_workflow.md` | Archivos de `/docs` del proyecto |
| Naming conventions | `2_architecture_rules.md` + `1_project_context.md` | Prefijos, sufijos, idioma |

---

## Checklist de replicación completo

```
# Pre-requisito Brownfield (omitir si es Greenfield)
[ ] Paso 0 ejecutado: diagnóstico del repositorio generado
[ ] Estado actual capturado en 1_project_context.md (no el ideal, el real)
[ ] Deuda técnica documentada en 2_architecture_rules.md como "aspiracional"

# Archivos del sistema
[ ] CLAUDE.md creado con comandos, arquitectura y convenciones
[ ] docs/prompts/ creada
[ ] 0_system_architect.md — router + contrato operativo
[ ] 1_project_context.md — identidad, módulos, glosario
[ ] 2_architecture_rules.md — reglas DEBE/NO DEBE (única fuente)
[ ] 3_development_workflow.md — flujo por bloques y gates
[ ] 4_documentation_workflow.md — propuestas documentales
[ ] 5_feature_generator.md — agente de features
[ ] 6_architecture_auditor.md — agente de auditoría
[ ] 7_refactor_assistant.md — agente de refactor
[ ] 8_prompt_evolution.md — gobernanza de prompts
[ ] 9_user_story_generator.md — agente de HU (opcional)
[ ] 10_ciber_seguridad_auditor.md — agente de seguridad (opcional)
[ ] docs/architecture.md — visión de capas del proyecto
[ ] docs/conventions.md — convenciones completas
[ ] docs/adr/ — carpeta para ADRs
```

---

## Anti-patrones comunes (errores a evitar)

### No duplicar reglas prescriptivas
Las reglas DEBE/NO DEBE solo van en `2_architecture_rules.md`. Si las copias en otros prompts, se crean contradicciones que confunden al asistente.

### No omitir el gate de auditoría
El sistema pierde su valor si se salta la auditoría arquitectónica. Cada bloque debe pasar por `6_architecture_auditor.md` antes de declararse listo.

### No mezclar idiomas en el dominio
Si decides que el dominio va en español, el glosario en `1_project_context.md` debe ser exhaustivo. El asistente usará esos términos al generar código.

### No definir reglas vagas
"El código debe ser limpio" no es una regla. "Application NO DEBE acceder directamente a DbContext" sí lo es. Las reglas deben ser verificables.

### No usar el mismo prompt para todo
El sistema funciona porque cada agente tiene un checklist específico. Si siempre le pides al System Architect que haga de todo, pierde el beneficio de la especialización.

### No forzar un alcance que no aplica al tipo de proyecto
Si tu proyecto es un frontend SPA, no necesitas reglas de entidades y agregados DDD. Si es una CLI, no necesitas una sección de "contrato API". Elimina las secciones irrelevantes al personalizar cada agente — un checklist con ítems que nunca aplican genera ruido y degrada la calidad de las respuestas.

### No documentar el estado ideal en un proyecto Brownfield
En un proyecto existente, `1_project_context.md` debe describir cómo funciona el código hoy. Si el código tiene deuda técnica, documéntala en `2_architecture_rules.md` como regla aspiracional. El asistente necesita saber la diferencia para no romper código funcional al intentar aplicar el patrón ideal.

---

## Evolución del sistema

Una vez en uso, el sistema se retroalimenta. El proceso recomendado:

1. **Detectar problema**: el asistente genera código que viola una regla.
2. **Identificar causa**: ¿qué regla falta o está ambigua?
3. **Determinar ownership**: ¿es una regla DEBE/NO DEBE (→ `2_architecture_rules.md`) o un flujo (→ `3_development_workflow.md`)?
4. **Aplicar el cambio mínimo**: sin reescribir todo el prompt.
5. **Verificar no duplicación** con el checklist de `8_prompt_evolution.md`.

El sistema es un artefacto vivo. Evoluciónalo cuando veas errores repetidos, no antes.

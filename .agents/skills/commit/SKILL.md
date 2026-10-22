---
name: commit
description: Genera un mensaje de commit en Conventional Commits considerando el impacto en release-please (versioning automático).
argument-hint: "[scope opcional]"
---

Genera un mensaje de commit para los cambios actuales siguiendo estas reglas:

## 1. Revisa los cambios

Ejecuta en paralelo:
- `git diff --staged` — cambios en staging
- `git diff` — cambios sin stagear
- `git log --oneline -5` — estilo de commits recientes del proyecto

## 2. Elige el tipo según el impacto en release-please

| Tipo | Versión | Aparece en CHANGELOG |
|------|---------|----------------------|
| `feat` | bump **minor** 1.2.0 → 1.3.0 | ✅ Features |
| `fix` | bump **patch** 1.2.0 → 1.2.1 | ✅ Bug Fixes |
| `refactor` | sin bump | ✅ Refactors |
| `perf` | sin bump | ✅ Performance |
| `docs` | sin bump | ✅ Documentation |
| `test`, `chore`, `ci`, etc. | sin bump | ❌ No aparece |
| `feat!` o footer `BREAKING CHANGE:` | bump **major** 1.0.0 → 2.0.0 | ✅ |

> Solo `feat` y `fix` (o breaking changes) disparan una nueva release automática al mergearse a `dev`.

## 3. Formato del mensaje

```
<tipo>(<scope>): <descripción en imperativo, minúsculas, sin punto final>

[cuerpo opcional si el cambio es complejo]

[BREAKING CHANGE: descripción, solo si aplica]
```

**PROHIBIDO**: no agregar `Co-Authored-By: Codex ...`, `Codex-Session: ...`, `🤖 Generated with Codex` ni ninguna firma/atribución de IA al mensaje de commit — ver "Git Commits & Pull Requests" en `AGENTS.md`. El mensaje termina en el `BREAKING CHANGE` (si aplica) o en la descripción/cuerpo, sin ningún footer adicional.

## 4. Respuesta

Muestra:
1. El mensaje de commit propuesto
2. Si **disparará o no una release** y por qué
3. Qué bump de versión resultaría (patch / minor / major) si aplica

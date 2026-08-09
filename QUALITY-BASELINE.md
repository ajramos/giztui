# QUALITY-BASELINE

Marca **ACTUAL** de complejidad del repositorio. Estos valores son
**umbrales de trinquete** (_ratchet_): la marca a **no superar**. No son
objetivos ni valores ideales — describen lo peor que ya existe hoy en el
código, para que a partir de aquí no empeore.

- **Fecha de medición:** 2026-08-09
- **Commit:** `1a89b93` (main)
- **Herramienta:** lizard 1.23.0 (`lizard . -s cyclomatic_complexity`)
- **Alcance:** código fuente Go + frontend TypeScript/React.

## Umbrales de trinquete (no superar)

| Métrica | Marca actual | Dónde vive hoy |
|---|---:|---|
| **CCN máximo (función)** | **265** | `runCommand` — `desktop/frontend/src/commandRunner.ts` (líneas 9–512) |
| **NLOC máximo (fichero)** | **3337** | `internal/tui/messages.go` |
| **Longitud máxima (función)** | **908** | `bindKeys` — `internal/tui/keys.go` (líneas 557–1464) |

Interpretación: mientras un cambio no genere una función con CCN > 265, ni un
fichero con NLOC > 3337, ni una función de más de 908 líneas, no se cruza la
marca actual. Cruzarla es un empeoramiento medible del peor caso del repo.

## Cómo reproducir

```sh
lizard . -s cyclomatic_complexity \
  -x "*/node_modules/*" -x "*/dist/*" -x "*/wailsjs/*" -x "*/mocks/*" \
  -x "*_test.go" -x "*.test.ts" -x "*.test.tsx" -x "*.spec.ts" -x "*.spec.tsx" \
  -x "*.d.ts" -x "*/e2e/*" -x "*/test-results/*" -x "*/build/*" -x "./test/*"
```

Exclusiones aplicadas: dependencias (`node_modules`), artefactos construidos
(`dist`, `build`), bindings generados (`wailsjs`), mocks generados
(`internal/services/mocks`, marcados `DO NOT EDIT`), y todo el código de test
(Go `*_test.go`, TS `*.test/*.spec`, `e2e/`, `test/`).

## Contexto de la medición (no forma parte del trinquete)

- Funciones analizadas: **4070** en **265** ficheros fuente.
- Funciones con CCN > 10: **220**.
- CCN medio global: **3.6**.

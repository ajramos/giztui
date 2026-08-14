# QUALITY-BASELINE

Marca **ACTUAL** de complejidad del repositorio. Estos valores son
**umbrales de trinquete** (_ratchet_): la marca a **no superar**. No son
objetivos ni valores ideales — describen lo peor que ya existe hoy en el
código, para que a partir de aquí no empeore.

- **Fecha de activación:** 2026-08-14
- **Base de código:** `bf0afe5` más las correcciones SDLC del plan #87
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

## Trinquete por fichero

`quality-baseline-per-file.csv` es el trinquete de grano fino: una fila por
fichero fuente con `max_nloc,max_ccn,max_func_len,funcs_over_ccn10`, ordenada
por ruta para diffs estables. Es más útil que el máximo global porque un
fichero que empeora no queda enmascarado por otro que ya ostenta el récord.
Regla: ninguna celda puede **subir** respecto a la fila registrada; puede
bajar libremente. `make ci-architecture` aplica la regla y también rechaza
ficheros fuente nuevos que todavía no hayan sido revisados y aceptados en el
baseline.

El baseline se actualizó al activar el gate para incorporar el trabajo de reglas
Gmail posterior a v1.27.0. A partir de este punto, `make
quality-baseline-update` solo debe ejecutarse deliberadamente después de revisar
el diff de métricas; no es un paso automático de CI.

Nota: `internal/tui/keys.go` queda registrado con `max_ccn=228` y
`max_func_len=908`, ambos de `bindKeys` (el `SetInputCapture` monolítico de
enrutado contextual), que es un problema aparte no abordado aquí. Congelarlo
no impide refactorizarlo: un trinquete de máximos solo prohíbe empeorar.

## Cómo verificar

```sh
make ci-tools
make ci-architecture
```

Exclusiones aplicadas: dependencias (`node_modules`), artefactos construidos
(`dist`, `build`), bindings generados (`wailsjs`), mocks generados
(`internal/services/mocks`, marcados `DO NOT EDIT`), y todo el código de test
(Go `*_test.go`, TS `*.test/*.spec`, `e2e/`, `test/`).

## Contexto de la medición

- Ficheros fuente bajo ratchet: **263**.
- Los máximos globales de la tabla superior no cambiaron al activar el gate.
- El ratchet congela deuda existente; los objetivos de reducción se gestionan
  por separado y no se obtienen regenerando el baseline al alza.

# Editions (theme plugins)

An **edition** is a complete, swappable front-end for Oriel. The host mounts
exactly one at a time; **Settings** changes it live and remembers the choice in
`localStorage`. The data/control layer is fixed; the presentation is a plugin.

Two ship in the box:

| id        | look                                                  |
| --------- | ----------------------------------------------------- |
| `studio`  | Clean, native-feel light/dark control panel (default) |
| `classic` | Calm teal dark control panel                          |

## The one rule

Editions talk to the backend **only** through the platform SDK
(`src/platform/index.js`):

```js
import { status, containers, invoke, lifecycle, fmt } from '../../platform/index.js'
```

If it isn't re-exported from there, it isn't contract — never reach into
`../lib/*` directly. An edition consumes reactive state and renders; the host
owns data fetching (one push-based live stream, no polling) and the global
overlays (command palette, confirm, toasts, op-progress). Keep behavior behind
the SDK so every edition benefits.

## Authoring guide

The full reference — the complete state/action/helper surface, adding a built-in
edition, runtime-loaded themes, and appearance/accents — lives in
[docs/THEMES.md](../../../docs/THEMES.md).

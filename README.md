Console JSON explorer written in Golang using tview

Navigate a json using a tree structure to inspect data.

Usage:
```bash
make run-demo
```

Use a different file:
```bash
make run JSON=path/to/file.json
```

Example JSON (`test.json`):
```json
{
  "name": "jex",
  "version": 1,
  "features": {
    "search": true,
    "tree": ["expand", "collapse"]
  }
}
```

Example UI (illustrative):
```text
┌──────────────────────── json explorer ─────────────────────────┐
│ { 3 }                                                          │
│ ├── name: "jex"                                                │
│ ├── version: 1                                                 │
│ └── features: { 2 }                                            │
│     ├── search: true                                           │
│     └── tree: [ 2 ]                                            │
│         ├── "expand"                                           │
│         └── "collapse"                                         │
└─────────────────────────────────────────────────────────────────┘
Search (/) [1/2] tree
Keys: / search  Enter toggle  n/p next/prev  e/c children  E/C all  q quit
```

Controls:
- `Enter`: Toggle expand/collapse on selected node
- `/`: Focus search field
- `Enter` (in search): Find first match and show index (`n/m`)
- `n` / `p`: Next/previous search match
- `e` / `c`: Expand/collapse children of selected node
- `E` / `C`: Expand/collapse entire tree
- `q`: Quit

Console JSON/XML TUI explorer written in Golang using tview

Navigate JSON or XML using a terminal user interface (TUI) tree to inspect data.

Usage:
```bash
make run-demo
```

Use a different file:
```bash
make run JSON=path/to/file.json
```

XML also works:
```bash
make run JSON=path/to/file.xml
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

Example XML:
```xml
<root version="1">
  <name>jex</name>
  <features>
    <search>true</search>
    <tree>expand</tree>
    <tree>collapse</tree>
  </features>
</root>
```

Example UI (illustrative):
```text
┌───────────────────────── jexplorer (JSON) ─────────────────────┐
│ { 3 }                                                          │
│ ├── name: "jex"                                                │
│ ├── version: 1                                                 │
│ └── features: { 2 }                                            │
│     ├── search: true                                           │
│     └── tree: [ 2 ]                                            │
│         ├── "expand"                                           │
│         └── "collapse"                                         │
└────────────────────────────────────────────────────────────────┘
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

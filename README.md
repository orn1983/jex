Console JSON/XML/YAML TUI explorer written in Golang using tview

Navigate JSON, XML, or YAML using a terminal user interface (TUI) tree to inspect data.

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

YAML works too:
```bash
make run JSON=path/to/file.yaml
```

Portable build (cross-compile):
```bash
make portable GOOS=linux GOARCH=amd64
```

Select architecture by setting `GOARCH` (and optionally `GOOS`):
- `amd64`: x86_64 CPUs (Intel/AMD 64-bit)
- `arm64`: Apple Silicon / ARM 64-bit
- `386`: 32-bit x86
- `arm`: 32-bit ARM

More examples:
```bash
make portable GOOS=darwin GOARCH=arm64
make portable GOOS=windows GOARCH=amd64
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
Keys: / search  Enter toggle  n/p next/prev  u/d page up/down  e/c children  E/C all  q quit
```

Controls:
- `Enter`: Toggle expand/collapse on selected node
- `/`: Focus search field
- `Enter` (in search): Find first match and show index (`n/m`)
- `n` / `p`: Next/previous search match
- `u` / `d`: Page up/down in the tree view
- `e` / `c`: Expand/collapse children of selected node
- `E` / `C`: Expand/collapse entire tree
- `q`: Quit

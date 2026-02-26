# howmanylines

Tiny Go CLI that counts total lines of code across all files in a project.


## Flags

- `-include-hidden` include hidden files and directories (default `true`)
- `-skip-dirs` extra comma-separated directory names to skip

By default it skips common generated/VCS directories like `.git`, `node_modules`, and `target`.

## Build From Source

```bash
go build -o howmanylines .
```
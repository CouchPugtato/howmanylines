# howmanylines

Tiny Go CLI that counts total lines of code across all files in a project.


## Run

```bash
howmanylines
howmanylines -count go,md
howmanylines -skip docs
```

## Flags

- `-count` file extensions to include
- `-skip` extra directory names to skip
- `-include-hidden` include hidden files and directories (default `true`)

By default it skips common generated/VCS directories like `.git`, `node_modules`, and `target`.

## Build From Source

```bash
go build -o howmanylines .
```

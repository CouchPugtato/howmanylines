# howmanylines

Tiny Go CLI that counts total lines of code in a project.


## Run

```bash
howmanylines
howmanylines -count go,md
howmanylines -skip docs
howmanylines -skip -include-hidden
```

## Flags

- `-count` file extensions to include
- `-skip` extra directory names to skip
- `-include-hidden` include hidden files and directories

By default it skips common generated/VCS directories like `.git`, `node_modules`, and `target`.

## Build From Source

```bash
git clone https://github.com/CouchPugtato/howmanylines
cd .\howmanylines\
go install
```

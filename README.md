# howmanylines

Tiny Go CLI that counts total lines of code in a project.


## Run

```bash
howmanylines
howmanylines -count go,md
howmanylines -skip docs
howmanylines -include-hidden
howmanylines -rank
howmanylines -rank 10
```

## Flags

- `-count` file extensions to include
- `-skip` extra directory names to skip
- `-include-hidden` include hidden files and directories (default `false`)
- `-rank` show both file and file extension leaderboards (defaults to top `3`)

By default it skips: 
- Common generated/VCS directories like `.git`, `node_modules`, `target` - files with no extension
- `.exe` files
- Likely binary/non-text files
- Common lock/manifest metadata files by default (ex. `package-lock.json`, `yarn.lock`, and Cargo lock/manifest files)

## Build From Source

```bash
git clone https://github.com/CouchPugtato/howmanylines
cd .\howmanylines\
go install
```

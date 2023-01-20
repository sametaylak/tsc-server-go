# tsc-server-go
## Idea
Having a tsc process running on the background and watch for the changes in given project root and emit possible errors found by compiler over TCP to any extensible code editor.

## Usage
```bash
go run . -rootFolderPath=<rootProjectFolderInAbsolutePath> -port=<port:9000>
```

After running tsc server, you can connect and listen localhost:port in your machine. You can find example response below
```json
[
    {
        "FileName": "src/foo.ts",
        "Line": 2,
        "Column": 4,
        "Message": "error TS1005: ')' expected."
    }
]
```
Response will be emitted only if there is a change on your project files.
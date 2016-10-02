# esh - easy SSH

### Build for macOS
#### Using Docker

- ```docker run -it --rm -v `pwd`:/go/src/esh -w /go/src/esh golang bash```
- `go get`
- `env GOOS=darwin GOARCH=386 go build -o bin/esh -v`

env GOOS=darwin GOARCH=386 go build -o ../bin/esh -v *.go

```bash
usage: esh [<flags>] <command> [<args> ...]

easy SSH

Flags:
  -h, --help  Show context-sensitive help (also try --help-long and --help-man).
```
###Commands:

###help \<command\>
Shows help for the specified command.

###add --name=NAME --server=127.0.0.1 --user=USER [<flags>]
Adds a SSH session to config.

###use <name>
Use a specific ssh session.

###list-all
List all saved SSH sessions.

###logout
Logout from current session.

###remove <name>
Remove a given session with name.

###get <getpath>
Get some file or folder.

###put <putpath>
Put some file or folder.

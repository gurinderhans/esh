# esh - easy SSH

### usage: esh [\<flags\>] \<command\> [\<args\> ...]

## Flags:
### -h, --help
Show context-sensitive help (also try --help-long and --help-man).

## Commands:

### help \<command\>
Shows help for the specified command.

### add --name=NAME --server=127.0.0.1 --user=USER [\<flags\>]
Adds a SSH session to config.

### use \<name\>
Use a specific ssh session.

### list-all
List all saved SSH sessions.

### logout
Logout from current session.

### remove \<name\>
Remove a given session with name.

### get \<getpath\>
Get some file or folder.

### put \<putpath\>
Put some file or folder.


# Build
### With Docker
```
cd $PROJECT_ROOT/src
docker run -it --rm -v `pwd`:/go/src/esh -w /go/src/esh golang bash
# In docker shell now...
go get
env GOOS=darwin GOARCH=386 go build -o ../bin/esh -v *.go
```

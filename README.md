# esh - easy SSH

### Build for macOS
Create a docker container for build env

```docker run -it --rm -v `pwd`:/go/src/esh -w /go/src/esh golang bash```

Build the app

`env GOOS=darwin GOARCH=386 go build -o bin/esh -v`

```bash
esh list-all - list all saved ssh servers
esh add -h <host> -u <user> -p <port> --key </path/to/key> # password will be asked if no key provided

esh use <name> - use this session
esh ls
esh pwd

#cheap way of downloading a single file
esh cat file.ext > file.ext

# SCP stuff
esh get <file|folder> // fetches into current local working dir
esh put <local_file|local_folder> <online_dir>
```

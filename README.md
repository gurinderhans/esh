# esh - easy SSH

### Any feedback, pull requests, etc are welcome!

Using **esh** you can interact with a remote machine without opening an interactive shell.

**esh** runs your command on a remote machine by reading your `args` and printing back `stdout` and `stderr`

# Preview
<img src="assets/esh.gif" alt="1" width=500>

# usage: esh [\<flags\>] \<command\> [\<args\> ...]
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
Logout from the current session.

### remove \<name\>
Remove a given session using a name.

### get \<getpath\>
Get some file or folder.

### put \<putpath\>
Put some file or folder.


# build()
### With Docker
```
cd $PROJECT_ROOT/src
docker run -it --rm -v `pwd`:/go/src/esh -w /go/src/esh golang bash
# In docker shell now...
go get
env GOOS=darwin GOARCH=386 go build -o ../bin/esh -v *.go
```

# future.print()
### TODOs (Bugs & Potential Features):
1. Fix any bugs, there are a few. The following is an example:
  - we get crashes if we try to upload/download files that don't exist (https://github.com/gurinderhans/esh/issues/1)

2. Optimize download and upload code to transfer files faster.
  - Using the `sftp` library has made the download speed exponentially slower than the upload speed. Downloading by opening a reverse SSH connection and then uploading from the server to the computer maybe be a potential solution, needs to be explored.
  
3. One way of potentially increasing the speed might be to create a separate background daemon program that keeps ssh connections alive and another main program that uses the 'open' ssh connections there to contact server/device. The effect is uncertain and needs to be tested before implementing.

4. Opening a `vim` on the local buffer and syncing it with the server so that the files are automatically saved in the server. This would involve fetching and putting of the file between the device and server behind the scenes. For e.g.
  - `esh vim /some/remote/path/to/file`

5. Open to other directions and suggestions.

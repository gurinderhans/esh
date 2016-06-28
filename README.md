# esh - easy SSH

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

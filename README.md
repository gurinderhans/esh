# esh - easy SSH

```bash
esh list-all - list all saved ssh servers
esh add -h <host> -u <user> -p <port> --key </path/to/key> # password will be asked if no key provided

esh use <name> - use this session
esh ls
esh pwd

# SCP stuff
esh get <file|folder> // fetches into current local working dir
esh put <local_file|local_folder> <online_dir>
```

# esh - easy SSH

```bash
esh list - list all saved ssh servers
esh add <user> <host> --key <file_path_to_key> # password will be asked if no key provided

esh <name> ls - list home directory of server or `/`
esh use <name> - use this session
NOW
esh ls = esh <name> ls
esh pwd

# SCP stuff
esh get <file|folder>
esh put <local_file|local_folder> <online_dir>
```

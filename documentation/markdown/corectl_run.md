## corectl run

Boots a new CoreOS instance

### Synopsis


Boots a new CoreOS instance

```
corectl run
```

### Options

```
      --cdrom string          append an CDROM (ISO) to VM
      --channel string        CoreOS channel (default "alpha")
      --cloud_config string   cloud-config file location (either an URL or a local path)
  -c, --cpus int              VM's vCPUS (default 1)
  -m, --memory int            VM's RAM, in MB, per instance (1024 < memory < 8192) (default 1024)
  -n, --name string           names the VM. (if unset defaults to VM's UUID)
  -o, --offline               doesn't go online to check for newer images than the locally available ones unless there is none available.
  -r, --root string           append a (persistent) root volume to VM
      --sshkey string         VM's default ssh key
  -u, --uuid string           VM's UUID (default "random")
      --version string        CoreOS version (default "latest")
      --volume value          append disk volumes to VM (default [])
```

### Options inherited from parent commands

```
      --debug   adds additional verbosity, and options, directed at debugging purposes and power users
```

### SEE ALSO
* [corectl](corectl.md)	 - CoreOS over OS X made simple.


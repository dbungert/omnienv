
# omnienv

omnienv aims to make it simpler to create and use container-based (or vm)
virtual environments, and to make a directory tree available to that virtual
environment.

omnienv is implemented using LXD and built with golang.

omnienv mounts $HOME read-only in the environment, along with read-write
mounting the working directory of the project being manged.

omnienv arranges for a matching $USER to be created in the environment with
passwordless sudo access, the shell invocation has an appropriate user
environment, and the resulting shell has a pty for greater compatibility with
various terminal applications.

omnienv shells can handle startup of containers and virtual-machines, including
handling the non-instant wait time for a LXD vm to go from stopped to
shell-ready.

## installation

    go install github.com/dbungert/omnienv/cmd/oe@ or something IDK

## usage

1. Identify a project directory you would like to associate with a container or
   vm.
2. Add a file `.omnienv.yaml` to this directory with the following contents:
```
system: noble
```
3. Run `oe --launch`
4. Standard lxd management commands can be used with the container.  For
   instance, this container can be deleted with `lxc delete foo-noble`, where
   `foo` is the basename of the directory containing .omnienv.yaml.

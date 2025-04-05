
# omnienv

omnienv aims to make it simpler to create and use container-based (or vm)
virtual environments, and to make a directory tree available to that virtual
environment.

omnienv is implemented using LXD and built with golang.

omnienv mounts $HOME read-only in the environment, along with read-write
mounting the working directory of the project being manged.

omnienv arranges for a matching $USER to be created in the environment with
passwordless sudo access, the shell invocation has appropriate user environment
variables set up, and the resulting shell has a pty for compatibility with
various terminal applications.

omnienv shells can handle waiting for startup of containers and
virtual-machines, including handling the non-instant wait time for a LXD vm to
go from stopped to shell-ready.  If the environment is stopped when a shell is
requested, omnienv first transparently starts that environment and initiates
the shell when possible.

## project status

Pre-alpha.  Config file format under active development and expected to make
breaking changes.

## installation

    go install github.com/dbungert/omnienv/cmd/oe@latest

## usage

1. Identify a project directory you would like to associate with a container or
   vm.
2. Add a file `.omnienv.yaml` to this directory with the following contents:
```
system: noble
```
3. Run `oe --launch`.  The container will be created, the project directory
   mounted into that environment, and the shell will start in the same
   directory you are in right now (but in the environment).
4. Standard lxd management commands can be used with the container.  For
   instance, this container can be deleted with `lxc delete foo-noble`, where
   `foo` is the basename of the directory containing `.omnienv.yaml`.

## config file format

An omnienv project is defined by the `.omnienv.yaml` config file and location.
The parent directory of that config is the working directory, and that working
directory is mounted read-write in the environment.

These fields are supported:
* `system`: the OS version of environment to use.  At this time only Ubuntu is
  supported, and only using the series names, so `jammy` for Ubuntu 22.04 and
  `noble` for Ubuntu 24.04 and so on.
* `virtualization` (optional): use a `container` (default) or `vm`.
* `label` (optional): the prefix for the environment name, this is inferred
  from the basename of the `rootdir` config.
* `rootdir` (optional): which directory to mount read-write in the environment.
  If unspecified, this is set to the parent directory of `.omnienv.yaml`.

## expected project direction

* The config file format is under active work, and I don't like some of the
  terms used just yet.
* Support expected for Ubuntu based on version numbers, and other Linux
  distributions handled by the lxd `images` remote.
* Ubuntu pre-bionic doesn't launch correctly today.

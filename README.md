# clueFS — a tool for tracing I/O activity at the file system level

## Overview
`cluefs` is a lightweight utility to collect data on the I/O events induced by an application when interacting with a file system. It emits detailed, machine-parseable data on every file system-level operation.

The trace information emitted by this utility is meant to be analysed using tools not included in this package.

## Motivation
The main goal of developing this utility is to observe and quantify the file I/O load induced by the software system being developed by the [LSST data management](http://dm.lsst.org/) team to process the data to be collected by the [Large Synoptic Survey Telescope](http://www.lsst.org/) (LSST).

However, `cluefs` does not depend on LSST software system and can be used in several unrelated contexts. It may also be useful for other use cases, such as to get an overall understanding of how file systems work or to observe the (usually hidden and unexpected) operations performed when you mount a file system on your computer. 

Although there are several tools for tracing system activity such as [strace](http://en.wikipedia.org/wiki/Strace), [DTrace](http://en.wikipedia.org/wiki/DTrace), [SystemTap](https://sourceware.org/systemtap/wiki) or [sysdig](http://www.sysdig.org/), for different reasons none of them was considered suitable for our particular use case.

## How to use
Let's suppose you want to observe what file operations the command `cat $HOME/data/hello.txt` induces on the file system where the argument file is located. You can use `cluefs` to expose the contents under the directory `$HOME/data` (the *shadow* directory) through a synthetized file system mounted on say `/tmp/trace`. To mount the file system use the command:

```bash
$ cluefs --shadow=$HOME/data  --mount=/tmp/trace &
```
Once the file system is successfully mounted, when an application accesses a file or directory under `/tmp/trace`, `cluefs` emits an event for every call to the file system (e.g. `access`, `open`, `read`, `close`, etc.). For instance, the command:

```bash
$ cat /tmp/trace/hello.txt
```
will make `cluefs` emit the events below (one event per line):

```
...
2015-03-23T10:26:35.839367864Z,2015-03-23T10:26:35.839794442Z,426578,fabio,9986,lsst,1021,/usr/bin/cat,23161,/home/fabio/data/hello.txt,stat
2015-03-23T10:26:35.840322045Z,2015-03-23T10:26:35.840364156Z,42111,fabio,9986,lsst,1021,/usr/bin/cat,23161,/home/fabio/data/hello.txt,openfile,O_RDONLY,0000
2015-03-23T10:26:35.840556082Z,2015-03-23T10:26:35.840572507Z,16425,fabio,9986,lsst,1021,/usr/bin/cat,23161,/home/fabio/data/hello.txt,read,15,0,4096,15
2015-03-23T10:26:35.841009818Z,2015-03-23T10:26:35.901634332Z,60624514,fabio,9986,lsst,1021,/usr/bin/cat,23161,/home/fabio/data/hello.txt,flush
2015-03-23T10:26:35.90204842Z,2015-03-23T10:26:35.902054482Z,6062,root,0,root,0,,0,/home/fabio/data/hello.txt,release
...
```

To get detailed help on how to use this utility, including examples of usage, do:

```bash
$ cluefs

USAGE:
   cluefs --mount=<directory>  --shadow=<directory>  [--out=<file>]
           [(--csv | --json)]  [--ro]
   cluefs --help
   cluefs --version

Use 'cluefs --help' to get detailed information about options and
examples of usage.
```

When you are done collecting the trace information you want, you can unmount the file system created by `cluefs` with the command:

```bash
$ sudo umount /tmp/trace
```


## Event formats

`cluefs` emits event records formatted in CSV or JSON. The format of each record is [documented here](doc/EventFormats.md).


## How to install

### Operating environment
This utility is tested on [Scientific Linux](https://www.scientificlinux.org/) v6 and v7, [Ubuntu](http://www.ubuntu.com/) v14.04, [CentOS](http://www.centos.org) v7  and [MacOS X](https://www.apple.com/osx/) v10.9. It is possible `cluefs` also works on other systems or other versions of those operating systems where its dependencies are satisfied (see below).

### Dependencies
To use `cluefs` you need *Filesystem in Userspace (FUSE)* installed on your system. To to that, please follow the installation instructions for your operating system according in the table below:

| To install FUSE on ...  | ... follow the instructions below |
| ----------------- |  ------------ |
| Ubuntu         |  `$ sudo apt-get --yes install fuse` |
| Scientific Linux, CentOS    |  `$ sudo yum install --assumeyes fuse` |
| MacOS X        |   install the latest stable version of [FUSE for OS X](https://osxfuse.github.io/) |

In addition, if you intend to build this software from sources you need both:

* the [Go programming language](https://golang.org/) tool chain, and
* a C compiler.

To install the Go tool chain please follow these [detailed instructions](http://golang.org/doc/install). To install a C compiler please refer to the table below:

| To install C compiler on ...  | ... follow the instructions below |
| ----------------- |  ------------ |
| Ubuntu         |  `$ sudo apt-get --yes install gcc` |
| Scientific Linux, CentOS    |  `$ sudo yum install --assumeyes gcc` |
| MacOS X        |  download and install [Xcode](https://developer.apple.com/xcode/downloads/), including its command line tools|

### Installation
The recommended way to install this tool is to download one of the ready-to-use binary files available for your target execution platform. Those are self-contained executable files so you only need to download, unpack and you are ready to start using the tool.

[**Download binary releases here**](https://github.com/airnandez/cluefs/releases).

Alternatively, to **build from sources** do:

```
go get -u github.com/airnandez/cluefs
```

## How this utility works
`cluefs` implements a synthesized file system which exposes all the files and directories existing on the underlying *shadow* file system. It intercepts each system call (e.g. `open`, `read`, etc.), emits a trace event about the call and forwards the operation to the appropriate file system for execution.`cluefs` collects the result of the operation and returns it to the calling application.

Although special attention has been given to make this utility as lightweight as possible, it is not intended to be permanently run in heavy-load I/O environments as there is an intrinsic non-zero performance penalty.

## Known limitations
Currently, lock-related file system operations are not supported by `cluefs`. That is, it does not emit traces for those operations and makes them appear as unsupported by the file system. These are the operations induced by calling the `fcntl(3)` file system call using as second argument any of the values `F_GETLK`, `F_SETLK` or `F_SETLKW`.

## You can contribute
Your contribution is more than welcome. There are several ways you can help:

* Test this software on your particular environment and let us know how it works. If it does not work for you and you think it should, please provide all the relevant details when [opening a new issue](https://github.com/airnandez/cluefs/issues)
* If you find a bug, please report it by [opening an issue](https://github.com/airnandez/cluefs/issues)
* If you spot a defect either in this documentation or in the source code documentation we consider it a bug so [please let us know](https://github.com/airnandez/cluefs/issues)
* Providing feedback on how to improve this software [by opening an issue](https://github.com/airnandez/cluefs/issues)

## Roadmap
The items in our to-do list are [documented separately](doc/ToDo.md).


## Disclaimer
Although we have payed a lot attention to make this utility as reliable as possible, it is still experimental and surely contains undiscovered bugs that may adversely affect your data.

In particular, please note that `cluefs` does **not** protect you against any destructive operation you can normally perform on your data. Use it at your own risk.

## Credits

### Author
This software was developed and is maintained by Fabio Hernandez at [IN2P3 / CNRS computing center](http://cc.in2p3.fr) (Lyon, France). 

### Acknowledgements

This work is based in other people's work, including:

* The [Go programming language](https://golang.org/) developement team,
* The very nice [Go FUSE file system library](http://bazil.org/fuse/)

## License
Copyright 2015 Fabio Hernandez

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

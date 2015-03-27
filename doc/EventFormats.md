# Trace event record formats emitted by `cluefs`

## Introduction

This document presents the format of each event record emitted by `cluefs`. Although event records include information generic to all of them (such as user id, group id, process id, etc.), each file system operation requires specific input parameters which are contained in the record. This means that the format of each record depends on the type of system call it refers to.

`cluefs` emits event records in CSV or JSON format. Unlike events in CSV format, event records in JSON format are self-described.

Below you will find the information emitted for each operation type. Every time stamp is given in UTC formated following [RFC3339](https://www.ietf.org/rfc/rfc3339.txt) with nanoseconds precision, for instance `2015-03-23T10:05:48.615390733Z`.

### CSV format — information common to all records
All the events emitted by `cluefs` in CSV format have the common set of values shown below. They are presented as found in every record, that is, from left to right:

1. start time stamp *(string, RFC3339)*
* end time stamp *(string, RFC3339)*
* operation duration *(integer, nanoseconds)*
* user name *(string)*
* user id *(integer)*
* group name *(string)*
* group id *(integer)*
* process executable path *(string)*
* process *(integer)*
* path of file/directory the operation targets *(string)*
* type of object named by path *(string, possible values: `"file"`, `"dir"`)*

Example CSV values common to all event records:

```csv
2015-03-23T10:05:48.615390733Z,2015-03-23T10:05:48.615422757Z,32024,fabio,9986,lsst,1021,/usr/bin/bash,22902,/home/fabio/data,dir
```

### JSON format — information common to all records
Every event emitted by `cluefs` in JSON format includes a `hdr` object which has the form shown below:

```json
"hdr":{
	"start":"2015-03-23T10:05:48.615390733Z", // start time stamp
	"end":"2015-03-23T10:05:48.615422757Z",   // end time stamp
	"nselaps": 32024,                         // operation duration (nanoseconds)
	"uid": 9986,                              // user id
	"usr":"fabio",                            // user name
	"gid": 1021,                              // group id
	"grp":"lsst",                             // group name
	"pid": 22902,                             // process id
	"proc":"/usr/bin/bash"                    // process executable path
},
```

In this document, records in JSON are shown in pretty print format for readability purposes. `cluefs` emits one record per line in compact form.


## Event formats
Click on the links below to get more details on the event format for the corresponding system call:

- [`access(2)`](#access)
- [`close(2)`](#close)
- [`creat(2)`](#creat)
- [flush](#flush)
- [`getxattr(2)`](#getxattr)
- [`listxattr(2)`](#listxattr)
- [`mkdir(2)`](#mkdir)
- [`open(2)`](#open)
- [`read(2)`](#read)
- [`readdir(3)`](#readdir)
- [`readlink(2)`](#readlink)
- [`removexattr(2)`](#removexattr)
- [`rename(2)`](#rename)
- [setattr](#setattr)  [***ToDo***]
- [`setxattr(2)`](#setxattr)
- [`stat(2)`](#stat)
- [`statfs(2)`](#statfs)
- [`symlink(2)`](#symlink)
- [`unlink(2)`](#unlink)
- [`write(2)`](#write)

_______________________________________________________________________
## access
An event of this type is emitted when an application calls the `access(2)` system call.

##### Example CSV record:
```
2015-03-23T10:05:48.615390733Z,2015-03-23T10:05:48.615422757Z,32024,fabio,9986,lsst,1021,/usr/bin/bash,22902,/home/fabio/data,dir,access,X_OK
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"access",
		"path":"/home/fabio/data",
		"isdir": true,
		"mode":"X_OK"
	}
}
```

##### Description of values specific to this operation:

* operation type: `access`
* path of file/directory this operation acts upon
* is the path a directory?
* access mode: possible values are ```F_OK```, ```X_OK```, ```W_OK```, ```R_OK```

_______________________________________________________________________
## close
An event of this type is emitted when an application calls the `close(2)` system call on an open file or directory. An event of this type usually follows a `flush` event.

##### Example CSV record:
```
2015-03-23T10:05:50.754493103Z,2015-03-23T10:05:50.754503307Z,10204,root,0,root,0,,0,/home/fabio/data/hello.txt,file,close
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"close",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false
	}
}
```

##### Description of values specific to this operation::

* operation type: `close`
* path of file or directory this operation acts upon
* is this path a directory?

_______________________________________________________________________
## creat
An event of this type is emitted when an application calls the `creat(2)` system call.

##### Example CSV record:
```
2015-03-26T11:23:30.622824963Z,2015-03-26T11:23:30.622847352Z,22389,fabio,9986,lsst,1021,/usr/bin/cp,14884,/home/fabio/data/hello.txt,file,creat,O_WRONLY|O_CREAT|O_EXCL,0644
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type": "creat",
		"path": "/home/fabio/data/hello.txt",
		"isdir": false,
		"flags": "O_WRONLY|O_CREAT|O_EXCL",
		"perm": "0644"
	}
}
```

##### Description of values specific to this operation::

* operation type: `creat `
* path of file or directory this operation acts upon
* is this path a directory?
* flags: possible values are combinations of `O_RDONLY`, `O_WRONLY`, `O_RDWR`, `O_CREAT`, `O_EXCL`, `O_TRUNC`, `O_APPEND`, `O_SYNC`
* permissions (in octal)

_______________________________________________________________________
## flush
An event of this type is emitted when an application calls the `close(2)` system call on an open file. An event of type `close` usually follows.

##### Example CSV record:
```
2015-03-26T11:23:30.623721516Z,2015-03-26T11:23:30.693056569Z,69335053,fabio,9986,lsst,1021,/usr/bin/cp,14884,/home/fabio/data/hello.txt,file,flush
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"flush",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false
	}
}
```

##### Description of values specific to this operation:

* operation type: `flush`
* path of file this operation acts upon
* is the path a directory?

_______________________________________________________________________
## getxattr
An event of this type is emitted when an application calls the `getxattr(2)` system call.

##### Example CSV record:
```
2015-03-26T11:23:30.43956521Z,2015-03-26T11:23:30.439571041Z,5831,fabio,9986,lsst,1021,/usr/bin/bash,14861,/home/fabio/data/hello.txt,file,getxattr,security.capability
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"getxattr",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"name":"security.capability"
	}
}
```

##### Description of values specific to this operation:

* operation type: `getxattr`
* path of the file or directory this operation acts upon
* is the path a directory?
* name of the extended attribute which value is requested

_______________________________________________________________________
## listxattr
An event of this type is emitted when an application calls the `listxattr(2)` system call.

##### Example CSV record:
```
2015-03-26T11:23:30.610836054Z,2015-03-26T11:23:30.610843728Z,7674,fabio,9986,lsst,1021,/usr/bin/attr,14878,/home/fabio/data/hello.txt,file,listxattr,65536
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"listxattr",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"size": 65536
	}
}
```

##### Description of values specific to this operation:

* operation type: `listxattr`
* path of the file or directory this operation acts upon
* is the path a directory?
* size of the buffer provided by the caller application


_______________________________________________________________________
## mkdir
An event of this type is emitted when an application calls the `mkdir(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.168675393Z,2015-03-26T13:41:15.16870229Z,26897,fabio,9986,lsst,1021,/usr/bin/mkdir,15479,/home/fabio/data/mydir,dir,mkdir,0755
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"mkdir",
		"path":"/home/fabio/data/mydir",
		"isdir": true,
		"mode":"0755"
	}
}
```

##### Description of values specific to this operation:

* operation type: `readdir`
* path of directory this operation acts upon
* is the path a directory?
* permissions (in octal)

_______________________________________________________________________
## open
An event of this type is emitted when an application calls the `open(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.025077899Z,2015-03-26T13:41:15.02510926Z,31361,fabio,9986,lsst,1021,/usr/bin/bash,15457,/home/fabio/data/hello.txt,file,open,O_WRONLY|O_APPEND,0001
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"open",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"flags":"O_WRONLY|O_APPEND",
		"perm":"0001"
	}
}
```

##### Description of values specific to this operation:

* operation type: `open`
* path of the file this operation acts upon
* is the path a directory?
* open flags: possible values are combinations of `O_RDONLY`, `O_WRONLY`, `O_RDWR`, `O_CREAT`, `O_EXCL`, `O_TRUNC`, `O_APPEND`, `O_SYNC`
* permissions (in octal)

_______________________________________________________________________
## read
An event of this type is emitted when an application calls the `read(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.117910671Z,2015-03-26T13:41:15.117919662Z,8991,fabio,9986,lsst,1021,/usr/bin/cat,15472,/home/fabio/data/hello.txt,file,read,36,0,4096,36
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"read",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"filesize": 36,
		"position": 0,
		"bytesreq": 4096,
		"bytesread": 36
	}
}
```

##### Description of values specific to this operation:

* operation type: `read`
* path of directory this operation acts upon
* is the path a directory?
* file size (in bytes)
* position within the file where the read operation is requested to start
* number of bytes requested
* number of bytes actually read

_______________________________________________________________________
## readdir
An event of this type is emitted when an application calls the `readdir(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.171066715Z,2015-03-26T13:41:15.171090152Z,23437,fabio,9986,lsst,1021,/usr/bin/ls,15480,/home/fabio/data,dir,readdir
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"readdir",
		"path":"/home/fabio/data",
		"isdir": true
	}
}
```

##### Description of values specific to this operation:

* operation type: `readdir`
* path of directory this operation acts upon
* is the path a directory?


_______________________________________________________________________
## readlink
An event of this type is emitted when an application calls the `readlink(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.171454767Z,2015-03-26T13:41:15.171460882Z,6115,fabio,9986,lsst,1021,/usr/bin/ls,15480,/home/fabio/data/mylink,file,readlink
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"readlink",
		"path":"/home/fabio/data/mylink",
		"isdir": false
	}
}
```

##### Description of values specific to this operation:

* operation type: `readlink`
* path of directory this operation acts upon
* is the path a directory?

_______________________________________________________________________
## removexattr
An event of this type is emitted when an application calls the `removexattr(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.165516485Z,2015-03-26T13:41:15.165527803Z,11318,fabio,9986,lsst,1021,/usr/bin/attr,15477,/home/fabio/data/hello.txt,file,removexattr,user.test.example.org
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"removexattr",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"name":"user.test.example.org"
	}
}
```

##### Description of values specific to this operation:

* operation type: `removexattr`
* path of the file or directory this operation acts upon
* is the path a directory?
* name of the extended attribute to be removed

_______________________________________________________________________
## rename
An event of this type is emitted when an application calls the `rename(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.285487273Z,2015-03-26T13:41:15.28550402Z,16747,fabio,9986,lsst,1021,/usr/bin/mv,15482,/home/fabio/data/hello.txt.copy,file,rename,/home/fabio/data/newfile.txt
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"close",
      	"old": "/home/fabio/data/hello.txt.copy",
  		"isdir": false,
      	"new": "/home/fabio/data/newfile.txt"
	}
}
```

##### Description of values specific to this operation::

* operation type: `rename`
* path of file or directory this operation acts upon
* is this path a directory?
* new name of to set to this file or directory

_______________________________________________________________________
## setxattr
An event of this type is emitted when an application calls the `setxattr(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.161938939Z,2015-03-26T13:41:15.161951342Z,12403,fabio,9986,lsst,1021,/usr/bin/attr,15474,/home/fabio/data/hello.txt,file,setxattr,user.test.example.org
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"setxattr",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"name":"user.test.example.org"
	}
}
```

##### Description of values specific to this operation:

* operation type: `setxattr `
* path of the file or directory this operation acts upon
* is the path a directory?
* name of the extended attribute to be set

_______________________________________________________________________
## stat
An event of this type is emitted when an application calls the `stat(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.166624309Z,2015-03-26T13:41:15.166632852Z,8543,fabio,9986,lsst,1021,/usr/bin/ln,15478,/home/fabio/data/mylink,file,stat
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"stat",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false
	}
}
```

##### Description of values specific to this operation:

* operation type: `stat`
* path of file or directory this operation acts upon
* is the path a directory?

_______________________________________________________________________
## statfs
An event of this type is emitted when an application calls the `statfs(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.166624309Z,2015-03-26T13:41:15.166632852Z,8543,fabio,9986,lsst,1021,/usr/bin/df,15478,/home/fabio/trace,dir,statfs
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"statfs",
		"path":"/home/fabio/trace",
		"isdir": false
	}
}
```

##### Description of values specific to this operation:

* operation type: `statfs `
* path of file or directory this operation acts upon
* is the path a directory?

_______________________________________________________________________
## symlink
An event of this type is emitted when an application calls the `symlink(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:15.167264101Z,2015-03-26T13:41:15.167290789Z,26688,fabio,9986,lsst,1021,/usr/bin/ln,15478,/home/fabio/data/mylink,file,symlink,/home/fabio/trace/hello.txt
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"symlink",
		"path":"/home/fabio/data/mylink",
		"isdir": false,
		"target":"/home/fabio/trace/hello.txt"
	}
}
```

##### Description of values specific to this operation:

* operation type: `symlink`
* path of the symbolic link to be created
* is the path a directory?
* path of the target file or directory the new symbolic link points to

_______________________________________________________________________
## unlink
An event of this type is emitted when an application calls the `unlink(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:14.877714466Z,2015-03-26T13:41:15.024393866Z,146679400,fabio,9986,lsst,1021,/usr/bin/bash,15457,/home/fabio/data/mylink,file,unlink
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"unlink",
		"path": "/home/fabio/data/mylink",
		"isdir": false
	}
}
```

##### Description of values specific to this operation::

* operation type: `unlink`
* path of file or directory this operation acts upon
* is this path a directory?


_______________________________________________________________________

## write
An event of this type is emitted when an application calls the `write(2)` system call.

##### Example CSV record:
```
2015-03-26T13:41:14.877412036Z,2015-03-26T13:41:14.877434607Z,22571,fabio,9986,lsst,1021,/usr/bin/bash,15457,/home/fabio/data/hello.txt,file,write,0,15,15
```

##### Example JSON record:
```json
{
	"hdr":{
		// ... common header ...
	},
	"op":{
		"type":"write",
		"path":"/home/fabio/data/hello.txt",
		"isdir": false,
		"position": 0,
		"bytesreq": 15,
		"byteswritten": 15
	}
}
```

##### Description of values specific to this operation:

* operation type: `write`
* path of file this operation acts upon
* is the path a directory?
* position within the file where this operation starts
* number of bytes requested to be written
* number of bytes actually written

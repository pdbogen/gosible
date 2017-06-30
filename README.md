# Components

## gosible

Gosible is my go-based configuration management tool. As you might guess, it's modeled on ansible.

## Testbed

A docker container designed to mimic a target server; for local testing to avoid mutable manipulation of target servers.

The "testbed" target of the Makefile will build and run the testbed locally, exposing its SSH as localhost:10022.

## Payload

The payload is a collection of YAML files, as they are easily readable. They are modeled after ansible tasks, etc., in concept, but are not intended to be 
at all compatible.

# Gosible Documentation

## Building

This package layout follows Golang best practices, meaning that the file you are currently reading should be present at `$GOPATH/src/github.com/pdbogen/gosible`.

If this is the case, the included Makefile will build gosible: Simply run `make gosible`, assuming that you have go 1.6+ available.

## Introduction

Gosible is a configuration management system, intended to support a variety of independent, declarative modules. The current implementation provides four real modules:

  * file
  * cmd
  * package
  * survey

Gosible operates on Targets, which are remote hosts that gosible can access with one of its transports (currently, just SSH).

Gosible performs operations called Tasks. Each Task specifies exactly one Module, which is something like one command to run or one file to create. Tasks 
may be grouped into Sets, and these named Sets can be called by Targets or even by other Sets.

Gosible implements a simple internal state registry for Tasks, and so can record the outcome of a Task and make execution decisions based on past recorded 
outcomes.

## Sets Definition

sets.yml is a YAML list, which means it has a top-level syntax like:

```
- <set>
- <set>
- <set>
```

Each Set is a two-member object:
```
- name: set-name
  tasks: <task-list>
```

A task-list is a list of tasks, each of which may have a name; and exactly one module (see Modules, below) and the parameters for the module, thus:

```
- name: set-name
  tasks:
  - name: first-task
    <module1-name>:
      <module1-param1>: <module1-param1-value>
  - name: second task
    <module2-name>:
      <module2-param1>: <module2-param1-value>
      <module2-param2>: <module2-param2-value>
```

### Example

```
- name: install nano
  tasks:
  - package:
      install: nano
- name: wreck the system
  tasks:
  - name: ruin /etc/shadow
    file:
      literal: this will go poorly
      dest: /etc/shadow
  - name: who needs bash
    file:
      literal: this is the end
      dest: /bin/bash
      mode: 0000
      
```

## Targets Definitions

Targets look an awful lot like sets, but they have a few more important parameters:

```
- name: target1
  address: 1.2.3.4
  port: 1234
  user: root
  credentialName: root-password
  tasks:
    <same as sets>
```

(In fact, an implicit Set is created for each target when it's run.)

## Credentials Definitions

Notice the `credentialName` field in the targets; this refers by name to an entry from the `credentials.yml` file, which has the following structure:

```
- name: root-password
  value: foobarbaz
```

## Sub-Componenents

There are four packages within gosible that are of concern:

* Core implements the core logic that loads credentials, targets, and tasks; and executes modules on targets using credentials. One such module is "task", which executes lists of modules, which may be named tasks. God help you if you execute a loop.
* Modules are native implementations of things we do to targets.
* Transports are ways that we do modules to targets.
* Types contains the primitives Credential, Target, Task, and Set.

## Modules

### Common Parameters
Two common parameters are available to control execution flow:

* `register` -- Set to the name of a variable; when a module annotated with `register` performs changes, that variable will be set to true, so...
* `when` -- set to a simple boolean expression composed of `register` variable names, 'and', 'or', and '!'; a module annotated with `when` will run only when the condition is true.
  * Example: `foo and !bar or baz`, which is the same as `(foo and !bar) or baz` (except that parentheses are actually not supported)

### Cmd

Cmd executes commands on the target host. Note that this interacts with 
`register` in an interesting fashion: Register will be set only if the command 
returns success (exit value 0); not set otherwise.

#### Parameters
* `cmd` -- The command to run. This is passed as an argument to `sh -c`.

### File

The basic file writer module.

#### Parameters
* `src` -- a path on the local filesystem, relative to the working directory the tool is run from
* `literal` -- the literal content of the file to write
* `dest` -- the path on the remote filesystem to write
* `mode` -- the octal mode to set on the file
* `uid` -- the uid that should own the file
* `gid` -- the gid that should own the file

`src` and `literal` are mutually exclusive.

### Package

Add and remove packages via apt-get.

#### Parameters
* `install` -- a space-separated list of packages that will be installed
* `remove` -- a space-separated list of packages that will be removed.

Package lists will be checked to ensure they do not contain contradictions.

### Set

From the user's perspective, Task is a module, even though the implementation 
is a little different. This module references task sets (defined in 
tasks.yml), and allows the user to compose reusable components.

#### Parameters
* `name` -- The name of the task set to run.

#### Example

```
- name: a_target
  address: 1.2.3.4
  credentialName: some_credential
  tasks:
  - set:
    name: a_set_referenced_by_name
```

### Survey

More of a hello world; but intended to be used to populate metadata. Is an "Always" module, which means it runs without being asked.

# Predicted Issues

There are a few obvious things that I've chosen not to handle right now:

* Explicit cleanup on premature termination
* Transactional changes (which would imply modules that are reversible, really)
* Non-password-based SSH
* Any other transport (though architectural support is there)
* SSH reconnects (for now, anyway)
* Native systemd / upstart modules

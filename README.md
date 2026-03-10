# Serve

> [!WARNING]
> This is a toy project. It was built entirely from a smartphone using Claude Code as an experiment in AI-assisted development. Do not run it outside of a sandboxed environment.

## What is it

Serve is a small HTTP file server written in Go. It lets you serve local files and directories over HTTP, with the ability to dynamically mount and unmount paths at runtime through a control API, no restart needed.

It runs two servers: a public one (default port 8080) that serves your files, and a control server (default port 8081) that accepts mount/unmount/list commands.

## Getting started

Requires Go 1.24 or later.

```
go install github.com/tigerwill90/serve@latest
```

## Usage

Start the server:

```
serve start
```

By default, the file server listens on `127.0.0.1:8080` and the control API on port `8081`. You can change this with flags:

```
serve start --host 0.0.0.0 --port 9090 --control-port 9091
```

Mount a local directory or file on a route:

```
serve mount ./public /static
serve mount ./config.json /config
```

List active mounts:

```
serve list
```

Unmount a route:

```
serve unmount /static
```

## License

MIT

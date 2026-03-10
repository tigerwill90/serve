# Serve

**Warning: This is a toy project.** The entire codebase was written using Claude Code on a smartphone, without ever opening a code editor. It exists as an experiment to see how far you can push AI-assisted development from a phone. The code has not been manually reviewed or audited. Do not trust it, do not run it outside of a sandboxed environment, and definitely do not run it on your personal or work machine.

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

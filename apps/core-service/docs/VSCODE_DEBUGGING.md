# Debugging in VS Code — launch.json

## The decision

The app is debugged through VS Code's built-in Go debugger (the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) driving [Delve](https://github.com/go-delve/delve)) via [.vscode/launch.json](../.vscode/launch.json), instead of `fmt.Println`/`log.Println` statements or attaching a debugger by hand from the terminal. Breakpoints, step-through, and variable inspection work the same way whether you're debugging `registerUser`, a `sqlc` query result, or a token-signing bug — no code changes needed to add or remove logging.

## Prerequisites — do this once per machine

1. **Go extension installed** in VS Code (`golang.go`).
2. **Delve (`dlv`) installed** — this is the actual debugger; the VS Code extension just drives it:
   ```
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```
   This installs to `$(go env GOPATH)/bin` (typically `~/go/bin`). That directory must be on your shell's `PATH`, or VS Code's Go extension can't find it and launching fails immediately with *"Couldn't find dlv at the Go tools path"*.
3. **`app.env` present** at the project root ([apps/core-service/app.env](../app.env)) with real values for `DB_DRIVER`, `DB_SOURCE`, `SERVER_ADDRESS`, `JWT_SECRET_KEY`, `JWT_ACCESS_TOKEN_DURATION` — see [CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md). Config is loaded from the process's current working directory, not from the source file's directory, which is why `cwd` matters in the config below.
4. **Postgres reachable** at whatever `DB_SOURCE` points to. The server dials the database during startup (`sql.Open` + the store layer's first query), before it ever binds a port, so debugging a `nil`-pointer in a handler still requires a working database connection first.

## Structure of launch.json

```jsonc
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug core-service",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/server",
            "cwd": "${workspaceFolder}"
        },
        {
            "name": "Debug current file",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${fileDirname}",
            "cwd": "${workspaceFolder}"
        },
        {
            "name": "Connect to server",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}",
            "port": 2345,
            "host": "127.0.0.1"
        }
    ]
}
```

Each entry is a distinct, independently selectable debug target — pick one from the dropdown next to the Run and Debug ▷ button (or `F5` runs whichever was last selected).

**`Debug core-service`** — the one you want for day-to-day work. Three fields matter:

- `"request": "launch"` — VS Code starts the process itself (compiles with debug symbols and runs it under `dlv`), as opposed to `"attach"`, which connects to a process already running.
- `"program": "${workspaceFolder}/cmd/server"` — points at the `main` package directory, not a file. This mirrors `go run ./cmd/server` in the [Makefile](../Makefile)'s `server` target — there are two `main.go` files in this repo (`cmd/server`, `cmd/migration`), so the path has to be explicit rather than inferred.
- `"cwd": "${workspaceFolder}"` — this is the field that's easy to get wrong. `config.LoadConfig(".")` (see [internal/config/config.go](../internal/config/config.go)) resolves `"."` against the debugger's working directory, *not* the location of `main.go`. Without this, `dlv` defaults `cwd` to the program's directory (`cmd/server`), Viper looks for `app.env` there, doesn't find it, and the app fails at the `LoadConfig` call before a single breakpoint can hit.

**`Debug current file`** — a generic fallback using `"mode": "auto"` and `"program": "${fileDirname}"`, which resolves to whatever directory the file you currently have open lives in. Useful for debugging a `_test.go` file or a small standalone package in isolation, but *not* reliable for running the whole server — if you have, say, `internal/api/auth_router.go` focused when you launch this, VS Go extension will try to treat that directory as the entry point, which only works if the open file happens to sit in a `main` package directory.

**`Connect to server`** — `"request": "attach"` with `"mode": "remote"` connects to a `dlv` process that is *already running and listening*, rather than starting a new one. This is for debugging a process you started separately — most commonly inside a Docker container or a remote host — by launching it there with:
```
dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./server
```
`port` and `host` must match wherever that headless `dlv` is actually listening, and `remotePath` tells VS Code how to map the remote source tree back to your local one for setting breakpoints — if the container's source isn't at the same relative structure as `${workspaceFolder}`, breakpoints silently won't bind.

## Edge cases — what happens when it goes wrong

**"Couldn't find dlv at the Go tools path"** — `dlv` isn't installed, or isn't on `PATH` for the environment VS Code launched in (a fresh shell PATH set in `.zshrc` isn't automatically picked up by an already-open VS Code window). Install it (see Prerequisites) and reload the VS Code window if it still isn't found — the extension caches the Go tool locations at startup.

**Process exits immediately with `cannot load config: ...`** — `cwd` isn't pointed at the directory containing `app.env`, or `app.env` doesn't exist there yet (only `app.env.example` does, e.g. right after cloning). Fix `cwd` in the config, or copy `app.env.example` to `app.env` and fill in real values.

**Process exits immediately with `cannot connect to db: ...`** — `DB_SOURCE` in `app.env` points at a Postgres instance that isn't running or isn't reachable from your machine (wrong host/port, container not started, VPN not connected). This fails before the HTTP server binds a port, so no amount of breakpoints in `internal/api` will help — fix the database connection first.

**`listen tcp :8080: bind: address already in use`** (or whatever `SERVER_ADDRESS` is) — a previous debug session's process is still running. Debug sessions started with `"request": "launch"` are usually killed when you stop debugging in VS Code, but a crashed extension host or a manually-started `go run ./cmd/server` in a terminal can leave the port held. Find and kill it (`lsof -i :8080` on macOS/Linux) or change `SERVER_ADDRESS`.

**Breakpoints show as hollow/unbound (a gray ring, not solid red)** — the running binary doesn't match the source at that line, almost always because you edited the file after the debug session started, or a stale build was used. Stop and restart the debug session; `"request": "launch"` always rebuilds first, so this is more common with `"attach"` against a binary someone built earlier.

**`Connect to server` hangs on "Connecting..." or fails immediately** — either nothing is listening on `host:port` yet (the remote `dlv --headless` process hasn't started, or started on a different port), or a firewall/Docker port mapping isn't forwarding that port to your machine. Verify with `nc -zv <host> <port>` before assuming the launch.json config is wrong.

**Env vars set in your shell don't seem to apply** — `viper.AutomaticEnv()` makes real environment variables override `app.env` (see [CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md)), but VS Code's debug launcher does **not** inherit your terminal's exported variables unless you add them explicitly via an `"env"` block in the configuration — a debug session and a terminal session are separate processes with separate environments.

## Adding a new debug configuration — the actual workflow

1. Open [.vscode/launch.json](../.vscode/launch.json) (Run and Debug ▷ → "create a launch.json file" if it doesn't exist yet, or the gear icon if it does).
2. Add a new object to the `configurations` array — don't edit an existing one in place if you still need the old behavior; give the new entry its own `"name"`.
3. Set `"program"` to the `main` package directory you want to run (a directory, not a `.go` file) and `"cwd"` to wherever that program expects to find its working files (config, migrations, etc.) — for anything in this repo that calls `config.LoadConfig(".")`, that's `${workspaceFolder}`.
4. If the program needs environment variables the debug process wouldn't otherwise have, add an `"env"` map of key/value pairs to the configuration rather than relying on your shell.

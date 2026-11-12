# flakes.nix

Development tools packaged as a Nix flake and kept up to date automatically.
Every hour a GitHub Actions job checks upstream for new releases, bumps the
pinned versions and hashes, runs `nix flake check`, and pushes to `main`.

## Packages

| Package | Source |
| ------- | ------ |
| `go`    | Latest stable official Go binary release from go.dev |

Systems: `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`.

## Usage

```nix
{
  inputs.devtools.url = "github:stubbedev/flakes.nix";

  outputs = { nixpkgs, devtools, ... }: {
    devShells.x86_64-linux.default =
      let pkgs = nixpkgs.legacyPackages.x86_64-linux;
      in pkgs.mkShell {
        packages = [ devtools.packages.x86_64-linux.go ];
      };
  };
}
```

`overlays.default` adds every package to a nixpkgs instance under the same
name, so `pkgs.go` becomes this toolchain. nixpkgs' `buildGoModule` is built
with a versioned attribute (`go_1_27`), so the overlay does not change it. To
build a module with this toolchain:
`pkgs.buildGoModule.override { go = devtools.packages.${system}.go; }`.

Ad hoc: `nix shell github:stubbedev/flakes.nix#go`.

## Adding a package

Each directory under `packages/` becomes a package of the same name
(`lib/packages.nix` finds them; `modules/packages.nix` exposes them).

- `packages/<name>/package.nix` (required) is a `callPackage`-style
  derivation. It reads its version and per-system hashes from
  `packages/<name>/hashes.json` and sets `meta.platforms` so it is only exposed
  where it builds.
- `packages/<name>/update.nix` (optional) declares where updates come from:
  a function of `{ lib, systems }` returning `{ type = "<source>"; settings = …; }`.
  `nix run .#update` passes it to the updater, which rewrites `hashes.json`.
  Seed a new package with `{"version": "0", "hashes": {}}`: it stays hidden on
  every system until the first update fills it in.

### The updater

The updater is a Go program (`cmd/update`, `internal/`, standard library only)
built by `modules/update.nix`, with its tests run by `nix flake check`. For each
package it loads the pin, asks the package's source for the latest release and
writes the new pin atomically. It refuses to downgrade, refuses a hash that
changed for an unchanged version, and keeps going when one package fails, then
exits non-zero. Each change is printed to stdout as one line, which becomes the
commit message body. Progress goes to stderr.

Sources live under `internal/source/` and are registered by `type` in
`cmd/update/main.go`:

| `type`       | Settings                                         | Source |
| ------------ | ------------------------------------------------ | ------ |
| `go-release` | `platforms`: Nix system → go.dev `<goos>-<goarch>` | `https://go.dev/dl/?mode=json` |

To support a new kind of upstream, add a package under `internal/source/`
implementing `update.Source` and register it in `sources`.

To add outputs that aren't packages (dev shells, checks, …), add a flake-parts
module under `modules/` and import it in `flake.nix`.

## Maintenance

```sh
nix run .#update    # bump all packages
nix flake check     # build every package for this system, run updater tests
nix fmt             # format Nix and Go
nix develop         # shell with this flake's Go and gopls
```

## Binary cache

CI pushes every package it builds (x86_64-linux, aarch64-linux, aarch64-darwin)
to the xilo cache at `https://nix.stubbe.dev/c/default/flakes`. The flake's
`nixConfig` offers it to `nix shell`/`nix build` run directly against this
flake. A consuming flake has to list it in its own `nixConfig` or `nix.conf`:

```
extra-substituters = https://nix.stubbe.dev/c/default/flakes
extra-trusted-public-keys = flakes:0Dr9MyPOvA6va2GVzflJcbrChNoZUeIZflyrww3O+YM=
```

The `XILO_TOKEN` repository secret holds a push token scoped to that cache.
CI builds `.#cache` (all packages for the runner's system linked together) and
pushes its closure.

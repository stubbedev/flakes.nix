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

Each directory under `packages/` becomes a package of the same name;
`modules/packages.nix` finds them automatically.

- `packages/<name>/package.nix` (required) is a `callPackage`-style
  derivation. Set `meta.platforms` so it is only exposed where it builds.
- `packages/<name>/update.nix` (optional) is a `callPackage`-style function
  that also receives `systems` (the flake's system list) and returns an
  executable. `nix run .#update` runs it from the repo root with
  `packages/<name>` as its only argument. It must rewrite the package's pinned
  data in place, print one `name: old -> new` line to stdout when it changes
  something, keep all other output on stderr, and exit non-zero on failure.

To add outputs that aren't packages (dev shells, checks, …), add a flake-parts
module under `modules/` and import it in `flake.nix`.

## Maintenance

```sh
nix run .#update    # bump all packages
nix flake check     # build every package for this system
nix fmt             # format
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

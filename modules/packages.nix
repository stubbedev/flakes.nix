{ lib, config, ... }:
let
  packagesDir = ../packages;

  names = lib.attrNames (
    lib.filterAttrs (_: type: type == "directory") (builtins.readDir packagesDir)
  );

  packageFile = name: packagesDir + "/${name}/package.nix";
  updaterFile = name: packagesDir + "/${name}/update.nix";

  updatableNames = lib.filter (name: builtins.pathExists (updaterFile name)) names;

  overlay = final: _: lib.genAttrs names (name: final.callPackage (packageFile name) { });
in
{
  flake.overlays.default = overlay;

  perSystem =
    { pkgs, ... }:
    let
      built = overlay pkgs pkgs;

      available = lib.filterAttrs (_: drv: lib.meta.availableOn pkgs.stdenv.hostPlatform drv) built;

      updaters = lib.genAttrs updatableNames (
        name: pkgs.callPackage (updaterFile name) { inherit (config) systems; }
      );

      update = pkgs.writeShellApplication {
        name = "update";
        runtimeInputs = [ pkgs.git ];
        text = ''
          cd "$(git rev-parse --show-toplevel)"
          ${lib.concatMapStrings (name: ''
            ${lib.getExe updaters.${name}} ${lib.escapeShellArg "packages/${name}"}
          '') updatableNames}
        '';
      };
    in
    {
      packages = available;
      checks = available;
      apps.update = {
        type = "app";
        program = lib.getExe update;
        meta.description = "Bump every package that ships an update.nix to its latest upstream release";
      };
    };
}

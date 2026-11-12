{ lib, config, ... }:
let
  packages = import ../lib/packages.nix { inherit lib; };
in
{
  perSystem =
    { pkgs, ... }:
    let
      updater = pkgs.buildGoModule {
        pname = "flakes-updater";
        version = "0";
        src = lib.fileset.toSource {
          root = ../.;
          fileset = lib.fileset.unions [
            ../go.mod
            ../cmd
            ../internal
          ];
        };
        vendorHash = null;
        env.CGO_ENABLED = 0;
        meta.mainProgram = "update";
      };

      configFile = pkgs.writeText "update-config.json" (
        builtins.toJSON {
          packages = map (name: {
            inherit name;
            pin = packages.pinPath name;
            source = import (packages.updateFile name) {
              inherit lib;
              inherit (config) systems;
            };
          }) packages.updatable;
        }
      );

      update =
        pkgs.runCommand "update"
          {
            nativeBuildInputs = [ pkgs.makeWrapper ];
            meta.mainProgram = "update";
          }
          ''
            makeWrapper ${lib.getExe updater} $out/bin/update --add-flags "-config ${configFile}"
          '';
    in
    {
      checks.updater = updater;
      apps.update = {
        type = "app";
        program = lib.getExe update;
        meta.description = "Bump every package that ships an update.nix to its latest upstream release";
      };
    };
}

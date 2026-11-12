{ lib }:
let
  root = ../packages;
  dir = name: root + "/${name}";
  names = lib.attrNames (lib.filterAttrs (_: type: type == "directory") (builtins.readDir root));
  updateFile = name: dir name + "/update.nix";
in
{
  inherit names updateFile;
  packageFile = name: dir name + "/package.nix";
  pinPath = name: "packages/${name}/hashes.json";
  updatable = lib.filter (name: builtins.pathExists (updateFile name)) names;
}

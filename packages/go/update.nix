{
  lib,
  writeShellApplication,
  writeText,
  curl,
  jq,
  coreutils,
  systems,
}:
let
  goPlatforms = lib.genAttrs systems (system: import ./platform.nix (lib.systems.elaborate system));
in
writeShellApplication {
  name = "update-go";
  runtimeInputs = [
    curl
    jq
    coreutils
  ];
  runtimeEnv.GO_PLATFORMS_FILE = writeText "go-platforms.json" (builtins.toJSON goPlatforms);
  text = builtins.readFile ./update.sh;
}

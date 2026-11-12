{ lib, systems }:
{
  type = "go-release";
  settings.platforms = lib.genAttrs systems (
    system: import ./platform.nix (lib.systems.elaborate system)
  );
}

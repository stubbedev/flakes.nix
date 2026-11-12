{ lib, pkgs }:
{
  type = "go-release";
  settings = {
    series = lib.versions.majorMinor pkgs.go_latest.version;
    files.source.kind = "source";
  };
}

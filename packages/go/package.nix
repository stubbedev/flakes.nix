{
  lib,
  stdenv,
  go_latest,
  fetchurl,
  callPackage,
  testers,
}:
let
  pin = lib.importJSON ./hashes.json;
  series = lib.versions.majorMinor pin.version;
  baseSeries = lib.versions.majorMinor go_latest.version;
in
assert lib.assertMsg (series == baseSeries)
  "go ${pin.version} needs a nixpkgs go_latest from the ${series} series to reuse its patches, but nixpkgs has ${go_latest.version}";
go_latest.overrideAttrs (
  finalAttrs: prev: {
    version = pin.version;
    src = fetchurl {
      url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
      hash = pin.hashes.source;
    };
    passthru = prev.passthru // {
      tests = {
        version = testers.testVersion {
          package = finalAttrs.finalPackage;
          command = "go version";
          version = "go${finalAttrs.version}";
        };
      }
      // lib.optionalAttrs stdenv.hostPlatform.isLinux {
        dynamic-import = callPackage ./tests/dynamic-import.nix { go = finalAttrs.finalPackage; };
      };
    };
  }
)

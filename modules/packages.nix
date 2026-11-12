{ lib, ... }:
let
  packages = import ../lib/packages.nix { inherit lib; };

  overlay =
    final: _: lib.genAttrs packages.names (name: final.callPackage (packages.packageFile name) { });
in
{
  flake.overlays.default = overlay;

  perSystem =
    { pkgs, ... }:
    let
      available = lib.filterAttrs (_: drv: lib.meta.availableOn pkgs.stdenv.hostPlatform drv) (
        overlay pkgs pkgs
      );
    in
    {
      packages = available;
      checks =
        available
        // lib.concatMapAttrs (
          name: drv: lib.mapAttrs' (test: lib.nameValuePair "${name}-${test}") (drv.passthru.tests or { })
        ) available;
      legacyPackages.cache = pkgs.linkFarm "flakes-cache" available;
    };
}

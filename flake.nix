{
  description = "Auto-updated development tool packages";

  nixConfig = {
    extra-substituters = [ "https://nix.stubbe.dev/c/default/flakes" ];
    extra-trusted-public-keys = [ "flakes:0Dr9MyPOvA6va2GVzflJcbrChNoZUeIZflyrww3O+YM=" ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs =
    inputs:
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [
        ./modules/packages.nix
        ./modules/update.nix
        ./modules/shell.nix
        ./modules/formatter.nix
      ];
    };
}

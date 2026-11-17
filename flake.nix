{
  description = "Auto-updated development tool packages";

  nixConfig = {
    extra-substituters = [ "https://xilos.dev/c/stubbedev/main" ];
    extra-trusted-public-keys = [ "xilos.dev/stubbedev/main-1:397R6S49v/FSvP+mZ3HlKrVN1KXhgzHIEIHxA/UuG6Y=" ];
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

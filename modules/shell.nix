{
  perSystem =
    { pkgs, self', ... }:
    {
      devShells.default = pkgs.mkShell {
        packages = [
          self'.packages.go
          pkgs.gopls
        ];
        env.GOTOOLCHAIN = "local";
      };
    };
}

{
  perSystem =
    { pkgs, ... }:
    {
      formatter = pkgs.nixfmt-tree.override {
        runtimeInputs = [ pkgs.go ];
        settings.formatter.gofmt = {
          command = "gofmt";
          options = [ "-w" ];
          includes = [ "*.go" ];
        };
      };
    };
}

{
  lib,
  buildGoModule,
  runCommand,
  stdenv,
  bintools,
  go,
}:
let
  app = (buildGoModule.override { inherit go; }) {
    pname = "go-dynamic-import";
    inherit (go) version;
    src = ./dynamic-import;
    vendorHash = null;
    env.CGO_ENABLED = 0;
    meta.mainProgram = "dynamic-import";
  };
  expected = lib.fileContents "${stdenv.cc}/nix-support/dynamic-linker";
in
runCommand "go-dynamic-import-interpreter" { } ''
  interpreter="$(${lib.getExe' bintools "readelf"} --string-dump=.interp ${lib.getExe app} | sed -n 's/^ *\[ *0\] *//p')"
  if [[ "$interpreter" != "${expected}" ]]; then
    echo "CGO_ENABLED=0 binary with a dynamic import has interpreter '$interpreter', want '${expected}'" >&2
    exit 1
  fi
  ${lib.getExe app}
  touch $out
''

{
  lib,
  stdenv,
  fetchurl,
}:
let
  inherit (lib.importJSON ./hashes.json) version hashes;
  inherit (stdenv.hostPlatform) system;
  goPlatform = import ./platform.nix stdenv.hostPlatform;
in
stdenv.mkDerivation {
  pname = "go";
  inherit version;

  src = fetchurl {
    url = "https://go.dev/dl/go${version}.${goPlatform}.tar.gz";
    hash = hashes.${system} or (throw "go: no hash for ${system} in hashes.json");
  };

  dontStrip = stdenv.hostPlatform.isDarwin;

  installPhase = ''
    runHook preInstall
    mkdir -p $out/share/go $out/bin
    cp -r . $out/share/go
    ln -s $out/share/go/bin/go $out/bin/go
    ln -s $out/share/go/bin/gofmt $out/bin/gofmt
    runHook postInstall
  '';

  inherit (stdenv.hostPlatform.go) GOOS GOARCH;
  CGO_ENABLED = 1;

  meta = {
    description = "Latest stable Go toolchain (official prebuilt binary)";
    homepage = "https://go.dev/";
    license = lib.licenses.bsd3;
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
    platforms = lib.attrNames hashes;
    mainProgram = "go";
  };
}

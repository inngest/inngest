let
  pkgs = import <nixos-23.05> { };
  unstable = import <nixpkgs> { };

  corepack = pkgs.stdenv.mkDerivation {
    name = "corepack";
    buildInputs = [ pkgs.nodejs-18_x ];
    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out/bin
      corepack enable --install-directory=$out/bin
    '';
  };

in pkgs.mkShell {
  nativeBuildInputs = [
    # Go
    unstable.go_1_21
    unstable.golangci-lint
    pkgs.gotests
    pkgs.gomodifytags
    pkgs.gore
    pkgs.gotools
    pkgs.gocode
    pkgs.protoc-gen-go

    # Lua
    pkgs.lua

    # Node
    # pkgs.yarn
    pkgs.nodejs-18_x

    # LSPs
    pkgs.gopls
    pkgs.nodePackages.typescript-language-server
    pkgs.nodePackages.vscode-json-languageserver
    pkgs.nodePackages.yaml-language-server
  ];

  packages = [ corepack ];
}

{ pkgs ? import (fetchTarball
  "https://github.com/NixOS/nixpkgs/archive/refs/tags/23.05.tar.gz") { } }:

with pkgs;

mkShell {
  buildInputs = [
    # Go
    pkgs.go
    pkgs.golangci-lint
    pkgs.gotests
    pkgs.gomodifytags
    pkgs.gore
    pkgs.gotools
    pkgs.gocode

    # LSPs
    pkgs.gopls

    # Tools
  ];
}

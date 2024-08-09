{
  description = "Inngest Dev Server";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        corepack = pkgs.stdenv.mkDerivation {
          name = "corepack";
          buildInputs = [ pkgs.nodejs_20 ];
          phases = [ "installPhase" ];
          installPhase = ''
            mkdir -p $out/bin
            corepack enable --install-directory=$out/bin
          '';
        };

      in {
        devShells.default = pkgs.mkShell {
          packages = [ corepack ];

          nativeBuildInputs = with pkgs; [
            # Go
            go
            golangci-lint
            gotests
            gomodifytags
            gore
            gotools
            protoc-gen-go
            goreleaser

            # Lua
            lua

            # Node
            typescript
            nodejs_20

            # LSPs
            gopls
            nodePackages.typescript-language-server
            nodePackages.vscode-json-languageserver
            nodePackages.yaml-language-server
            lua-language-server

            # Tools
            sqlite
            sqlc
            buf
            protoc-gen-go
            protoc-gen-connect-go
            natscli
            nats-server
            nats-top
          ];
        };
      });
}

{
  description = "Inngest Dev Server";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=master";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;

          config.allowUnfree = true;
        };
        corepack = pkgs.stdenv.mkDerivation {
          name = "corepack";
          buildInputs = [ pkgs.nodejs_22 ];
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
            goreleaser
            delve

            # Lua
            lua

            # Node
            typescript
            nodejs_22

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
            protoc-gen-go-grpc
            protoc-gen-connect-go
            claude-code
            gemini-cli
          ];

          shellHook = ''
            export GOBIN=$PWD/bin
          '';
        };
      });
}

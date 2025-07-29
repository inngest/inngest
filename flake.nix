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
      in {
        packages.default = pkgs.callPackage ./package.nix { shortCommit = self.dirtyShortRev or self.shortRev or "dirty"; };
        devShells.default = pkgs.mkShell {
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
            pnpm_8

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
        };
      });
}

{
  description = "Inngest Dev Server";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=master";
    flake-utils.url = "github:numtide/flake-utils";
    # Temporary until the next nixpkgs lock update provides golangci-lint 2.13.2.
    golangci-lint-override.url =
      "github:nixos/nixpkgs/0968519e14f7aa7d3e9b389682bd74d2b51c8ce8";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      golangci-lint-override,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;

          config.allowUnfree = true;
        };
        golangci-lint = golangci-lint-override.legacyPackages.${system}.golangci-lint;
        corepack = pkgs.stdenv.mkDerivation {
          name = "corepack";
          buildInputs = [ pkgs.nodejs_22 ];
          phases = [ "installPhase" ];
          installPhase = ''
            mkdir -p $out/bin
            corepack enable --install-directory=$out/bin
          '';
        };

        # Build grpc-gateway generators from the same revision used by go.mod.
        grpc-gateway = pkgs.buildGoModule rec {
          pname = "grpc-gateway";
          version = "2.28.0-strict-query-parameters-0bd4dcb0";

          src = pkgs.fetchFromGitHub {
            owner = "inngest";
            repo = "grpc-gateway";
            rev = "0bd4dcb02324be2078eed320514ec8eccc9087c6";
            hash = "sha256-79+J6A7mt8NgNSJtlL3Q/KNTY6QqlxBflgGcBl1OJtw=";
          };

          vendorHash = "sha256-jVP5zfFPfHeAEApKNJzZwuZLA+DjKgkL7m2DFG72UNs=";

          subPackages = [
            "protoc-gen-grpc-gateway"
            "protoc-gen-openapiv2"
          ];

          ldflags = [
            "-X main.version=v${version}"
            "-X main.commit=${src.rev}"
          ];
        };

      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            corepack
            grpc-gateway
          ];

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
            typescript-language-server
            vscode-json-languageserver
            yaml-language-server
            lua-language-server

            # Tools
            sqlite-interactive
            sqlc
            buf
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            protoc-gen-connect-go
            swagger-codegen3
            git-cliff
          ];

          shellHook = ''
            export GOBIN=$PWD/bin
            export PATH="$PATH:$GOBIN:$HOME/go/bin"
          '';
        };
      }
    );
}

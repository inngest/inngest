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

        # install grpc gateway binaries
        grpc-gateway = pkgs.stdenv.mkDerivation rec {
          pname = "grpc-gateway";
          version = "2.27.1";

          arch = if pkgs.stdenv.isAarch64 then "arm64" else "x86_64";
          os = if pkgs.stdenv.isDarwin then "darwin" else "linux";

          protoc-gen-grpc-gateway = pkgs.fetchurl {
            url =
              "https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v${version}/protoc-gen-grpc-gateway-v${version}-${os}-${arch}";
            sha256 = {
              "darwin-x86_64" =
                "0z53qg60mwax8kvrjcs5bngf46ksh86zdwib0pq0dbqz32n04n54";
              "darwin-arm64" =
                "0g6rnkl4sy5wd8iyxvli892y3d3228bsiia223p3md9nplsj1fba";
              "linux-x86_64" =
                "0jb6d53irbzkcmii0xaykc9m528zjja0inzl97g49mcp21qzidvl";
              "linux-arm64" =
                "1nmwjv8ymqm51q5sfkisx9vhkla5s03w14g69h33g6118rcq6zl3";
            }."${os}-${arch}";
          };

          protoc-gen-openapiv2 = pkgs.fetchurl {
            url =
              "https://github.com/grpc-ecosystem/grpc-gateway/releases/download/v${version}/protoc-gen-openapiv2-v${version}-${os}-${arch}";
            sha256 = {
              "darwin-x86_64" =
                "19msmjgwcpzk6ji2b1dycgkxahmcm89ws7l52ssw80akzqqnn3ga";
              "darwin-arm64" =
                "13zi36ghlhv6cdcx9xza6iy2sp8mcz8axg8r4jy3n6626fwf4wrv";
              "linux-x86_64" =
                "09r3dscpxnj487iinqjxy5psyh14vz7gclj4xl4w21hm1wzcaqyi";
              "linux-arm64" =
                "1162m9s7899ia2g3pmynb2403pdszpn91b6dasagzm4pl4gdn41m";
            }."${os}-${arch}";
          };

          dontUnpack = true;

          installPhase = ''
            mkdir -p $out/bin
            install -m755 ${protoc-gen-grpc-gateway} $out/bin/protoc-gen-grpc-gateway
            install -m755 ${protoc-gen-openapiv2} $out/bin/protoc-gen-openapiv2
          '';
        };

      in {
        devShells.default = pkgs.mkShell {
          packages = [ corepack grpc-gateway ];

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
            swagger-codegen3
          ];

          shellHook = ''
            export GOBIN=$PWD/bin
            export PATH="$PATH:$GOBIN:$HOME/go/bin"
          '';
        };
      });
}

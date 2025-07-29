{
  lib,
  buildGoModule,
  fetchFromGitHub,
  stdenv,
  pnpm_8,
  jq,
  writeScript,
  shortCommit,
}: let
  version = "dev";

  src = ./.;

  inngest-ui = stdenv.mkDerivation (finalAttrs: {
    inherit version src;
    pname = "inngest-ui";

    nativeBuildInputs = [
      pnpm_8
      pnpm_8.configHook
    ];

    buildPhase = ''
      cd ui/apps/dev-server-ui && pnpm build
    '';

    installPhase = ''
      mkdir -p $out
      cp -r dist $out
      cp -r .next/routes-manifest.json $out/dist
    '';

    pnpmWorkspaces = ["dev-server-ui"];

    pnpmDeps = pnpm_8.fetchDeps {
      inherit (finalAttrs) pname pnpmWorkspaces;
      inherit version src;
      hash = "sha256-FrG/Z2frOpDi/6hPunzbGxMJVrbXSfhKhI3VOE1JogM=";
      sourceRoot = "${finalAttrs.src.name}/ui";
    };
    pnpmRoot = "ui";
  });
in
  buildGoModule {
    inherit version src inngest-ui;
    pname = "inngest";

    vendorHash = null;

    preBuild = ''
      go generate ./...
      cp -r ${inngest-ui}/dist/* ./pkg/devserver/static
    '';

    postInstall = ''
      mv $out/bin/cmd $out/bin/inngest
    '';

    ldflags = [
      "-s -w"
      "-X github.com/inngest/inngest/pkg/inngest/version.Version=${version}"
      "-X github.com/inngest/inngest/pkg/inngest/version.Hash=${shortCommit}"
    ];

    # The Inngest CI/CD uses GoReleaser to build the package, and the env `CGO_ENABLED` is set in the configuration file for GoReleaser
    # https://github.com/inngest/inngest/blob/main/.goreleaser.yml#L9
    env.CGO_ENABLED = 0;

    subPackages = ["cmd/"];

    meta = {
      description = "Queuing and orchestration for modern software teams";
      homepage = "https://www.inngest.com/";
      license = lib.licenses.asl20;
      mainProgram = "inngest";
      platforms = lib.platforms.all;
    };
  }
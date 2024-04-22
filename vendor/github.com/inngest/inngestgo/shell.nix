let pkgs = import <nixos-23.11> { };

in pkgs.mkShell {
  nativeBuildInputs = [
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

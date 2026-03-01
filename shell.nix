let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-25.11";

  pkgs = import nixpkgs {
    config = { };
    overlays = [ ];
  };
in
pkgs.mkShell {
  hardeningDisable = [ "fortify" ];
  buildInputs = with pkgs; [
    go_1_25
    # libpcap
    # gcc
    golangci-lint
    goreleaser
    nilaway
    nodejs_24
    pnpm_10
    just
    just-lsp
    nil
    nixd
    govulncheck
    zellij
    air
    delve
    markdownlint-cli2
    sourcepawn-studio
  ];
}

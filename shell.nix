let
  # nixpkgs = fetchTarball {
  #   url = "https://github.com/NixOS/nixpkgs/archive/e0629618b4b419a47e2c8a3cab223e2a7f3a8f97.tar.gz";
  #   sha256 = "sha256-e0629618b4b419a47e2c8a3cab223e2a7f3a8f97";
  # };
  #nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-25.11";
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/e0629618b4b419a47e2c8a3cab223e2a7f3a8f97.tar.gz";

  pkgs = import nixpkgs {
    config = { };
    overlays = [ ];
  };
in
pkgs.mkShell {
  hardeningDisable = [ "fortify" ];
  buildInputs = with pkgs; [
    # libpcap
    # gcc
    golangci-lint
    goreleaser
    nilaway
    nodejs
    pnpm
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
    buf
    protoc-gen-go
    protoc-gen-connect-go
    oapi-codegen
    protobuf-language-server
    sql-formatter
    rcon-cli
  ];
}

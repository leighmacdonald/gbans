let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-25.11";
  #nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/e0629618b4b419a47e2c8a3cab223e2a7f3a8f97.tar.gz";

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
    go
    golangci-lint
    goreleaser
    nilaway
    nodejs
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
    buf
    protoc-gen-go
    protoc-gen-connect-go
    oapi-codegen
    sql-formatter
    protoc-gen-es
    protobuf-language-server
    rcon-cli
    clang-tools
    govulncheck
  ];
}

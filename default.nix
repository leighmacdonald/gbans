let
  # July 2
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/b5aa0fbd538984f6e3d201be0005b4463d8b09f8.tar.gz";

  pkgs = import nixpkgs {
    config = {
      hardeningDisable = [ "all" ];
    };
    overlays = [ ];
  };
in
pkgs.mkShell {
  shellHook = ''
    export PATH="$PWD/frontend/node_modules/.bin:$PATH"
  '';
  hardeningDisable = [ "fortify" ];
  buildInputs = with pkgs; [
    gcc
    go
    golangci-lint
    goreleaser
    nilaway
    nodejs
    pnpm_11
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
    pgcli
  ];
}

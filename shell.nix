let
  #nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-25.11";
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/ce657ac8a02003528e4ea4bb59d58e1c634b790c.tar.gz";

  pkgs = import nixpkgs {
    config = {   hardeningDisable = [ "all" ]; };
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

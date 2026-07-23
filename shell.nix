let
  # July 23
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/4c4fc8beef2dbd5813acf699dbded4d7a9c2e4c0.tar.gz";

  pkgs = import nixpkgs {
    config = {
      hardeningDisable = ["all"];
    };
    overlays = [];
  };
in
  pkgs.mkShell {
    shellHook = ''
      export PATH="$PWD/frontend/node_modules/.bin:$PATH"
    '';
    hardeningDisable = ["fortify"];
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
      typescript-go
      markdownlint-cli2
      sourcepawn-studio
      buf
      protoc-gen-go
      protoc-gen-connect-go
      oapi-codegen
      sql-formatter
      protoc-gen-es
      protobuf-language-server
      typescript-language-server
      rcon-cli
      clang-tools
      govulncheck
      pgcli
    ];
  }

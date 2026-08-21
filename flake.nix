{
  inputs = {
    # July 23
    nixpkgs.url = "github:NixOS/nixpkgs/4c4fc8beef2dbd5813acf699dbded4d7a9c2e4c0";
  };

  outputs =
    { self, nixpkgs }:
    let
      lib = nixpkgs.lib;

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      makeDevShell =
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config = {
              hardeningDisable = [ "all" ];
            };
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
            pgcli
          ];
        };
    in
    {
      devShells = lib.genAttrs systems (system: {
        default = makeDevShell system;
      });
    };
}

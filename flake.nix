{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nix-sourcemod.url = "github:leighmacdonald/nix-sourcemod";
  };

  outputs =
    { nixpkgs, nix-sourcemod, ... }:
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
            export PATH="${nix-sourcemod.packages.${system}.sourcemod_stable}/addons/sourcemod/scripting:$PATH"
          '';
          hardeningDisable = [ "fortify" ];
          buildInputs = with pkgs; [
            # core backend required deps
            gcc
            go_1_27
            buf
            protoc-gen-go
            protoc-gen-connect-go
            oapi-codegen

            # core frontend required deps
            nodejs
            pnpm_11
            protoc-gen-es
            typescript

            # go tooling
            golangci-lint
            goreleaser
            govulncheck
            delve

            # sourcemod tooling
            sourcepawn-studio
            rcon-cli
            clang-tools
            nix-sourcemod.packages.${system}.sourcemod_stable

            # misc tooling
            zellij
            air
            nilaway
            just
            just-lsp
            nil
            nixd
            markdownlint-cli2
            sql-formatter
            protobuf-language-server
            typescript-language-server
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

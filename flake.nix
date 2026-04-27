{
  description = "Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };

  outputs = { self, nixpkgs, ... }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [ "aarch64-linux" "x86_64-linux" "aarch64-darwin" "x86_64-darwin" ];
    in
    {
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          go = pkgs.go_1_24; # ← change toolchain version here
        in
        {
          default = pkgs.mkShell {
            packages = [
              go
              pkgs.gopls
              pkgs.gotools
              pkgs.delve        # debugger
              pkgs.golangci-lint
            ];

            # ✅ Evaluated at shell entry time — $HOME is real here
            shellHook = ''
              export GOPATH="''${GOPATH:-$HOME/go}"
              export GOROOT="${go}/share/go"
              export PATH="$GOPATH/bin:$PATH"

              echo "Go $(go version | cut -d' ' -f3) — GOPATH=$GOPATH"
            '';
          };
        });

      formatter = forAllSystems (system:
        nixpkgs.legacyPackages.${system}.nixpkgs-fmt
      );
    };
}

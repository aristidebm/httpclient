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
          go = pkgs.go_1_24;
        in
        {
          default = pkgs.mkShell {
            packages = [
              go
              pkgs.gopls
              pkgs.gotools
              pkgs.delve
              pkgs.golangci-lint
            ];
          };
        });

      formatter = forAllSystems (system:
        nixpkgs.legacyPackages.${system}.nixpkgs-fmt
      );
    };
}

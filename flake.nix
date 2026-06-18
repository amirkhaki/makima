{
  description = "makima - Personal assistant with rule-based automation";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.gopls
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "makima";
          version = "1.0.0";
          src = ./.;
          vendorHash = null;
          doCheck = true;
          checkPhase = ''
            go test ./...
          '';
        };
      }
    );
}

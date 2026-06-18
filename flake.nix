{
  description = "makima - Multiple binaries in one package";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        # Development shell with Go tools
        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
          ];
        };

        # Single package containing all binaries
        packages.default = pkgs.buildGoModule {
          pname = "makima";
          version = "1.0.0";
          src = ./.;

          vendorHash = null;  # ← replace after first `nix build`

        #   # Optional: if you have C dependencies, add buildInputs
        #   # buildInputs = [ pkgs.pkg-config ];
        };

        # Define each binary as an app so `nix run .#widget` works
        # apps = builtins.listToAttrs (map (name: {
        #   name = name;
        #   value = {
        #     type = "app";
        #     program = "${self.packages.${system}.default}/bin/${name}";
        #   };
        # }) cmdDirs);

        # If you still need the NixOS module from ./module.nix
        # nixosModules.default = import ./module.nix;
      }
    );
}

{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {inherit system;};

        app = pkgs.buildGoModule {
          pname = "deploys-cli";
          version = "1.1.0";

          src = ./.;

          vendorHash = "sha256-S5nq6DK4356LCMYKX3anjcySAxZhGxFWu1qKXR44C94=";

          meta = with pkgs.lib; {
            description = "Deploys.app CLI";
            license = licenses.mit;
          };
        };
      in {
        packages = {
          default = app;
          app = app;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_24
            gopls
            golangci-lint

            air

            pre-commit
          ];

          shellHook = ''
            echo "Go development environment ready"
            echo "Go version: $(go version)"
          '';
        };
      }
    );
}

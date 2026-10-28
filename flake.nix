{
  description = "Vector Charts CLI (vc)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        version = "0.1.0";
      in {
        packages.default = pkgs.buildGoModule {
          pname = "vc";
          inherit version;
          src = ./.;
          vendorHash = "sha256-QG4sBu0FdHdGnuinymDXB0FczD5VFeAnpJ9YqRj10lU=";
          subPackages = [ "cmd/vc" ];
          ldflags = [
            "-s" "-w"
            "-X github.com/zydro-marine/vector-charts-cli/internal/cmd.Version=${version}"
          ];
          env.CGO_ENABLED = "0";
          meta = with pkgs.lib; {
            description = "Vector Charts command-line interface";
            homepage = "https://github.com/zydro-marine/vector-charts-cli";
            license = licenses.unfree;
            mainProgram = "vc";
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/vc";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools goreleaser ];
        };
      });
}

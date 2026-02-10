{
  description = "Download wallpapers from wallhaven.cc";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "wallhaven_dl";
          version = self.shortRev or self.dirtyShortRev or "dev";

          src = ./.;

          vendorHash = "sha256-I+TYWPzipkyEv9vN5Tfpk026wLQ+dS/BvbGIoYSJLWM=";

          ldflags = [
            "-s" "-w"
            "-X main.Version=${self.shortRev or "dev"}"
          ];
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls go-task ];
        };
      }
    );
}

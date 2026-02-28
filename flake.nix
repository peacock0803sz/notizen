{
  description = "notizen; a simple note-taking CLI";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        formatter = pkgs.nixpkgs-fmt;
        flakedPkgs = pkgs;
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            uv
            go
          ];
        };
      }
    );
}

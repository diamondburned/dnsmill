{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }@inputs:

    (flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            self.formatter.${system}
          ];
        };

        packages = rec {
          default = dnsmill;
          dnsmill = pkgs.buildGoModule {
            pname = "dnsmill";
            version = self.rev or "unknown";
            src = self;

            vendorHash = "sha256-gT2FBsy9QVwJaS695TUYYUEUlAzdAedXnqzvciqPLCQ=";

            subPackages = [ "cmd/dnsmill" ];

            meta = {
              description = "Declaratively set your DNS records with dnsmill, powered by libdns.";
              homepage = "https://libdb.so/dnsmill";
              license = pkgs.lib.licenses.isc;
              mainProgram = "dnsmill";
            };
          };
        };

        formatter = pkgs.nixfmt-rfc-style;

        checks = import ./nix/checks.nix { inherit pkgs self; };
      }
    ))
    // {
      nixosModules = rec {
        default = dnsmill;
        dnsmill = import ./nix/module.nix inputs;
      };
    };
}

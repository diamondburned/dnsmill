{
	inputs = {
		nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
		flake-utils.url = "github:numtide/flake-utils";
	};

	outputs = { self, nixpkgs, flake-utils }:
		flake-utils.lib.eachDefaultSystem (system: let
			pkgs = nixpkgs.legacyPackages.${system};
		in
		{
			packages = rec {
				default = dnsmill;
				dnsmill = pkgs.buildGoModule {
					pname = "dnsmill";
					version = self.rev or "unknown";
					src = self;

					vendorHash = "sha256-kZPHwO/Hxe3cSmRqzq2d0ESf+V20T1miQ+ZMN21bs+g=";

					meta = {
						description = "Declaratively set your DNS records with dnsmill, powered by libdns.";
						homepage = "https://libdb.so/dnsmill";
						license = pkgs.lib.licenses.isc;
						mainProgram = "dnsmill";
					};
				};
			};
		});
}

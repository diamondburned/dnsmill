{ pkgs, self }:

{
  moduleConfig = pkgs.nixosTest {
    name = "moduleConfig";
    nodes.machine =
      { config, pkgs, ... }:
      {
        imports = [ self.nixosModules.dnsmill ];

        services.dnsmill = {
          enable = true;
          profiles."test" = {
            config = {
              duplicatePolicy = "overwrite";
            };
            providers = {
              cloudflare.zones = [ "libdb.so" ];
            };
            records = {
              "test1.libdb.so".hosts = "localhost";
              "test2.libdb.so".cname = "test1.libdb.so";
            };
          };
          environment = {
            CLOUDFLARE_API_TOKEN = "hi";
          };
        };
      };
    testScript =
      { nodes }:
      builtins.trace (builtins.toJSON nodes.machine.services.dnsmill.finalProfiles) ''
        machine.sleep(10)
      '';
  };
}

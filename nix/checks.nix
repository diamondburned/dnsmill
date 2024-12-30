{ pkgs, self }:

{
  moduleConfig = pkgs.nixosTest {
    name = "moduleConfig";
    nodes.machine =
      { ... }:
      {
        imports = [ self.nixosModules.dnsmill ];

        services.dnsmill = {
          profiles."test" = {
            config = {
              duplicatePolicy = "overwrite";
            };
            providers = {
              cloudflare = [ "libdb.so" ];
            };
            domains = {
              "libdb.so" = {
                dnsmill_test = "localhost";
              };
            };
          };
          environment = {
            CLOUDFLARE_API_TOKEN = "hi";
          };
        };
      };
    testScript =
      { ... }:
      ''
        machine.sleep(10)
      '';
  };
}

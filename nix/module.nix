{ self, ... }:

{
  config,
  pkgs,
  lib,
  ...
}:

with lib;
with builtins;

let
  profileType = types.submodule {
    options = {
      enable = mkOption {
        type = types.bool;
        default = true;
        description = ''
          Whether to enable the profile.
        '';
      };

      config = mkOption {
        type = profileConfigType;
        default = { };
        description = ''
          The profile configuration.
        '';
      };

      providers = mkOption {
        type = types.attrsOf (types.listOf types.string);
        example = {
          cloudflare = [ "libdb.so" ];
        };
        description = ''
          Map of providers to the domains they manage.
        '';
      };

      domains = mkOption {
        type = types.attrsOf (
          types.attrsOf (
            types.oneOf [
              (types.str)
              (types.listOf types.str)
              (types.submodule {
                options = {
                  hosts = {
                    type = types.oneOf [
                      # single IP/host
                      (types.str)
                      # multiple IPs/hosts
                      (types.listOf types.str)
                    ];
                    description = ''
                      Hosts indirectly represents A and AAAA records. The host addresses are
                      resolved into a list of IP addresses, with IPv4 addresses being handled as A
                      records and IPv6 addresses being handled as AAAA records.
                    '';
                  };
                };
              })
              (types.submodule {
                option = {
                  cname = {
                    type = types.str;
                    description = ''
                      CNAME represents a single CNAME record.
                    '';
                  };
                };
              })
            ]
          )
        );
        description = ''
          Either a list of IP addresses or a list of hostnames for A and AAAA records, or
          the proper submodule form of the config.
        '';
      };
    };
  };

  profileConfigType = types.submodule {
    options = {
      duplicatePolicy = mkOption {
        type = types.enum [
          "error"
          "overwrite"
        ];
        default = "error";
        description = ''
          DuplicatePolicy is the policy to apply when a duplicate DNS record is found.
          Options:
            - error returns an error if a duplicate DNS record is found.
              It is the safest policy and is the default.
            - overwrite overwrites the existing DNS record with the new one.
        '';
      };
    };
  };
in

{
  options.services.dnsmill = {
    enable = mkEnableOption "dnsmill";

    profiles = mkOption {
      type = types.attrsOf profileType;
      default = { };
      description = ''
        A map of profiles to their configuration.
      '';
    };

    finalProfiles = mkOption {
      type = types.attrsOf types.path;
      description = ''
        A map of profiles to their final configuration as JSON paths.
      '';
      readOnly = true;
    };

    environment = mkOption {
      type = types.attrsOf types.string;
      default = { };
      description = ''
        The environment variables to use for dnsmill.
      '';
    };

    environmentFile = mkOption {
      type = types.nullOr types.path;
      default = null;
      description = ''
        The environment file to use for dnsmill.
      '';
    };

    package = mkOption {
      type = types.package;
      default = self.packages.${pkgs.system}.dnsmill;
      description = ''
        The package to use for dnsmill.
      '';
    };
  };

  # Bruh.
  # https://gist.github.com/udf/4d9301bdc02ab38439fd64fbda06ea43

  # Configure only systemd.services.
  config = {
    services.dnsmill.finalProfiles = mapAttrs (
      profileName: profile:
      let
        finalJSON = builtins.toJSON (removeAttrs profile [ "enable" ]);
        configFile = pkgs.writeText "dnsmill-profile-${profileName}.json" finalJSON;
      in
      configFile
    ) config.services.dnsmill.profiles;

    systemd.services = mkIf config.services.dnsmill.enable (
      mkMerge (
        mapAttrsToList (
          (profileName: profile: {
            "dnsmill@${profileName}" = {
              enable = profile.enable;
              description = "dnsmill profile ${profileName}";
              environment = config.services.dnsmill.environment;
              serviceConfig = {
                Type = "oneshot";
                ExecStart = escapeShellArgs [
                  (getExe config.services.dnsmill.package)
                  "-f"
                  "json"
                  (config.services.dnsmill.finalProfiles.${profileName})
                ];
                Restart = "on-abnormal";
                RestartSec = 30;
                EnvironmentFile = config.services.dnsmill.environmentFile;
              };
              startLimitBurst = 5;
              startLimitIntervalSec = 5 * 60; # 5 minutes;
              after = [ "network.target" ];
              requires = [ "network.target" ];
              wantedBy = [ "multi-user.target" ];
            };
          })
        ) config.services.dnsmill.profiles
      )
    );
  };
}

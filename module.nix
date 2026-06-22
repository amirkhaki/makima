{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.services.makima;
in
{
  options.services.makima = {
    enable = mkEnableOption "makima personal assistant daemon";
  };

  config = mkIf cfg.enable {
    systemd.user.services.makima = {
      description = "Makima personal assistant daemon";
      serviceConfig = {
        ExecStart = "${pkgs.makima}/bin/makima daemon start";
        Restart = "on-failure";
        RestartSec = 5;
      };
      wantedBy = [ "default.target" ];
    };
  };
}

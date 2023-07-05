{ pkgs, ... }:

{
  # https://devenv.sh/basics/
  # env.GREET = "devenv";

  # https://devenv.sh/packages/
  packages = [ pkgs.git pkgs.reflex ];

  # https://devenv.sh/scripts/
  # scripts.hello.exec = "echo hello from $GREET";

  scripts.watch.exec = ''
    export BUILD_TYPE=DEV
    reflex -r '\.go$' -s -- sh -c 'go run .'
  '';

  enterShell = ''
    git --version
    go version
  '';

  # https://devenv.sh/languages/
  # languages.nix.enable = true;
  languages.go.enable = true;

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # https://devenv.sh/processes/
  # processes.ping.exec = "ping example.com";

  # See full reference at https://devenv.sh/reference/options/
}

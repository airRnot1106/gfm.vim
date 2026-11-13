{
  description = "Jump to a fuzzy matched word in Vim and Neovim on goda.vim";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    goda-vim = {
      url = "github:Shougo/goda.vim/v0.7.2";
      flake = false;
    };
    goda-ks = {
      url = "github:airRnot1106/goda-ks.vim";
      flake = false;
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      goda-vim,
      goda-ks,
    }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = "0.1.0";

      mkPackages = pkgs: rec {
        goda-fm = pkgs.buildGo126Module {
          pname = "goda-fm";
          inherit version;
          src = ./.;
          vendorHash = "sha256-ml58gta+1OsSX5EO+I5DWbPRGk3Z2uJnbAvD8ZThea0=";
          subPackages = [ "." ];
          doCheck = false;
          postInstall = ''
            mv $out/bin/goda-fm.vim $out/bin/goda-fm
          '';
          meta.mainProgram = "goda-fm";
        };

        goda-vim-plugin = pkgs.vimUtils.buildVimPlugin {
          pname = "goda.vim";
          version = "0-${goda-vim.shortRev}";
          src = goda-vim;
        };

        vim-plugin = pkgs.vimUtils.buildVimPlugin {
          pname = "goda-fm.vim";
          inherit version;
          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./LICENSE
              ./autoload
              ./doc
              ./plugin
            ];
          };
          dependencies = [ goda-vim-plugin ];
          preInstall = ''
            mkdir bin
            cp ${goda-fm}/bin/goda-fm bin/goda-fm
          '';
        };

        default = vim-plugin;
      };
    in
    {
      packages = forAllSystems (
        pkgs:
        let
          built = mkPackages pkgs;
        in
        {
          inherit (built) goda-fm vim-plugin default;
          goda-vim = built.goda-vim-plugin;
        }
      );

      checks = forAllSystems (
        pkgs:
        let
          built = mkPackages pkgs;
          dictionary = pkgs.fetchurl {
            url = "https://github.com/oguna/migemo-compact-dict-latest/releases/download/v0.2/migemo-compact-dict";
            hash = "sha256-+jrCQLM1iSIvzUCRDe0sFpc3TgGvFNwrtd32g6oumho=";
          };
          goTest =
            name: command:
            built.goda-fm.overrideAttrs {
              pname = "goda-fm-${name}";
              doCheck = true;
              checkPhase = ''
                runHook preCheck
                export GODA_KS_TEST_DICT=${dictionary}
                ${command}
                runHook postCheck
              '';
              installPhase = "touch $out";
              postInstall = "";
            };
        in
        {
          unit = goTest "unit" "go test ./internal/...";
          e2e = goTest "e2e" ''
            export HOME=$TMPDIR
            export PATH=${
              pkgs.lib.makeBinPath [
                pkgs.vim
                pkgs.neovim
              ]
            }:$PATH
            export GODA_FM_E2E_GODA=${goda-vim}
            export GODA_FM_E2E_GODA_KS=${goda-ks}
            export GODA_FM_E2E_STRICT=1
            go test -v -count=1 ./e2e
          '';
        }
      );

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_26
            pkgs.vim
            pkgs.neovim
          ];
        };
      });
    };
}

{
  description = "Jump to a fuzzy matched word in Vim and Neovim on goda.vim";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    goda-vim = {
      url = "github:Shougo/goda.vim/v0.7.2";
      flake = false;
    };
    # TODO: use the published repository, with the replace directive of go.mod removed.
    gks = {
      url = "git+file:///Users/airrnot/dev/github.com/airRnot1106/gks.vim";
      flake = false;
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      goda-vim,
      gks,
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
        gfm = pkgs.buildGo126Module {
          pname = "gfm";
          inherit version;
          src = ./.;
          vendorHash = "sha256-R9ZhCxUkZ1hHI6Ujnv3w0rSpfMDcN1VguY/TLHKjqI8=";
          subPackages = [ "." ];
          doCheck = false;
          postPatch = ''
            cp -r ${gks} ../gks.vim
            chmod -R u+w ../gks.vim
          '';
          postInstall = ''
            mv $out/bin/gfm.vim $out/bin/gfm
          '';
          meta.mainProgram = "gfm";
        };

        goda-vim-plugin = pkgs.vimUtils.buildVimPlugin {
          pname = "goda.vim";
          version = "0-${goda-vim.shortRev}";
          src = goda-vim;
        };

        vim-plugin = pkgs.vimUtils.buildVimPlugin {
          pname = "gfm.vim";
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
            cp ${gfm}/bin/gfm bin/gfm
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
          inherit (built) gfm vim-plugin default;
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
            built.gfm.overrideAttrs {
              pname = "gfm-${name}";
              doCheck = true;
              checkPhase = ''
                runHook preCheck
                export GKS_TEST_DICT=${dictionary}
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
            export GFM_E2E_GODA=${goda-vim}
            export GFM_E2E_GKS=${gks}
            export GFM_E2E_STRICT=1
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

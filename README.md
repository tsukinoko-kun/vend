# vend

Manage external sources easily

## Usage

Write a `vend.yaml` or generate it using `vend init`.
You can even port your Git Submodules to vend using `vend init --from-gitsubmodules` (this option is not 100% reliable)

```yaml
---
version: 1
scripts:
  ttf_windows: powershell.exe -File ./vendored/SDL_ttf/external/Get-GitModules.ps1
  ttf_unix: ./vendored/SDL_ttf/external/download.sh
sources:
  - url: https://github.com/libsdl-org/SDL.git
    reference_name: release-3.2.10
  - url: https://github.com/libsdl-org/SDL_image.git
    reference_name: release-3.2.4
  - url: https://github.com/libsdl-org/SDL_ttf.git
    reference_name: release-3.2.2
  - url: https://github.com/skypjack/entt.git
    reference_name: v3.15.0
  - url: https://github.com/tsukinoko-kun/ClayMan.git
    reference_name: v0.1.1
```

Install the dependencies in your `vend.yaml` using `vend sync`.
This will download all repositories into one of these directories (global `vend` directory):
- `$XDG_DATA_HOME/vend/` if `XDG_DATA_HOME` is set
- `~/Library/Application Support/vend` on macOS
- `~/.local/share/vend` on Linux
- `%APPDATA%\vend` on Windows
- `~/.vend` as fallback on other platforms

Then a symlink for each source will be created in your project `vendored` directory pointing to the downloaded repository in the global `vend` directory.

<br>

You can add a source using `vend add <url>@<ref_name>`.
`url` can be any http GIT url. SSH is currently not supported.
`ref_name` can be any valid Git reference name but a tag is recommended.

The `scripts` work like their counterparts in a package.json file.
Expansion of environment variables and argument parsing is in POSIX style.

## Update

`vend` can update itself using `vend update`.
This downloads the latest version of `vend` from GitHub and replaces the current executable with the new one.

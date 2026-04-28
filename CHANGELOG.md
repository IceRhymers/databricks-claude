# Changelog

## [0.13.0](https://github.com/IceRhymers/databricks-claude/compare/v0.12.1...v0.13.0) (2026-04-28)


### Features

* --credential-helper and --generate-desktop-config for Claude Desktop ([12ef44c](https://github.com/IceRhymers/databricks-claude/commit/12ef44cea5cd4577ce6c800cb3ffba6ad3f17b4f))
* add --credential-helper and --generate-desktop-config for Claude Desktop ([00e05c1](https://github.com/IceRhymers/databricks-claude/commit/00e05c1c85c61235129b39136cd3140664c21b8e)), closes [#92](https://github.com/IceRhymers/databricks-claude/issues/92)
* document release-please conventional commit requirements in AGENTS.md ([bd3e865](https://github.com/IceRhymers/databricks-claude/commit/bd3e865162f291a6f07549041306356547425da9))


### Bug Fixes

* address PR [#93](https://github.com/IceRhymers/databricks-claude/issues/93) review — generate both artifacts, MDM CLI pinning, README docs ([ba7c7b3](https://github.com/IceRhymers/databricks-claude/commit/ba7c7b386bf662d656a15058ec686a066aae9c83))
* make credential helper work from Claude Desktop's GUI subprocess ([c58b3f0](https://github.com/IceRhymers/databricks-claude/commit/c58b3f02d86e076b64d96d13b3191efbd30445c3))

## [0.12.1](https://github.com/IceRhymers/databricks-claude/compare/v0.12.0...v0.12.1) (2026-04-10)


### Bug Fixes

* scheme-aware health check and TLS arg forwarding in headless.Ensure ([#84](https://github.com/IceRhymers/databricks-claude/issues/84)) ([fd3b0aa](https://github.com/IceRhymers/databricks-claude/commit/fd3b0aaf0c13100abd8000c11348846444bf62da))

## [0.12.0](https://github.com/IceRhymers/databricks-claude/compare/v0.11.0...v0.12.0) (2026-04-10)


### Bug Fixes

* treat missing settings.json as empty document ([#81](https://github.com/IceRhymers/databricks-claude/issues/81)) ([e957c4f](https://github.com/IceRhymers/databricks-claude/commit/e957c4f0db558e797db1609878c111dacb2919c1))


### Refactoring

* extract shared proxy utilities into pkg/ ([#79](https://github.com/IceRhymers/databricks-claude/issues/79)) — extracts health, lifecycle, state, and headless packages from the top-level package; adds ListenerPort, PathForPort, and PrintUpdateNotice helpers; ~185 lines removed from main package with no behavior changes

## [0.11.0](https://github.com/IceRhymers/databricks-claude/compare/v0.10.1...v0.11.0) (2026-04-10)


### Features

* add pkg/updater — GitHub release checker with cache and numeric semver comparison ([#75](https://github.com/IceRhymers/databricks-claude/issues/75)) ([32e58cf](https://github.com/IceRhymers/databricks-claude/commit/32e58cf52fa7837f59283d4c921d9472f855dd63))
* add update subcommand and startup update check ([#77](https://github.com/IceRhymers/databricks-claude/issues/77)) ([0e9b525](https://github.com/IceRhymers/databricks-claude/commit/0e9b525b0da0cff2246daa12d685b4352192af57))

## [0.10.1](https://github.com/IceRhymers/databricks-claude/compare/v0.10.0...v0.10.1) (2026-04-09)


### Bug Fixes

* accept --shell=bash form from Homebrew generate_completions_from_executable ([#71](https://github.com/IceRhymers/databricks-claude/issues/71)) ([4ec2a08](https://github.com/IceRhymers/databricks-claude/commit/4ec2a082879d483b212f3bcf7e022535dc303c62))

## [0.10.0](https://github.com/IceRhymers/databricks-claude/compare/v0.9.1...v0.10.0) (2026-04-09)


### Features

* add --install-hooks, --uninstall-hooks, and proxy lifecycle hooks ([#63](https://github.com/IceRhymers/databricks-claude/issues/63)) ([1e2b84f](https://github.com/IceRhymers/databricks-claude/commit/1e2b84f2f008dab16bbe4eeee673f9a72a29a46d))
* add POST /shutdown endpoint and idle timeout for headless lifecycle management ([#61](https://github.com/IceRhymers/databricks-claude/issues/61)) ([fe184d1](https://github.com/IceRhymers/databricks-claude/commit/fe184d142019c0348793dd8a1538dd166c987557))
* add shell tab completions (bash/zsh/fish) ([#69](https://github.com/IceRhymers/databricks-claude/issues/69)) ([7b011e4](https://github.com/IceRhymers/databricks-claude/commit/7b011e471f05a320d1407372248ebe0d8adbdad7))


### Bug Fixes

* remove DATABRICKS_HOST and DATABRICKS_CONFIG_PROFILE from settings.json env ([#67](https://github.com/IceRhymers/databricks-claude/issues/67)) ([22649e7](https://github.com/IceRhymers/databricks-claude/commit/22649e7acfc2545803f844a99c85399da5f0689a))

## [0.9.1](https://github.com/IceRhymers/databricks-claude/compare/v0.9.0...v0.9.1) (2026-04-07)


### Bug Fixes

* normalize trailing newline in main.go ([#52](https://github.com/IceRhymers/databricks-claude/issues/52)) ([f2ae5cf](https://github.com/IceRhymers/databricks-claude/commit/f2ae5cfa1da9ab851388fdeb6b0ad5e24b9962d4))

## [0.9.0](https://github.com/IceRhymers/databricks-claude/compare/v0.8.1...v0.9.0) (2026-04-07)


### Features

* dispatch Homebrew formula update on release ([#48](https://github.com/IceRhymers/databricks-claude/issues/48)) ([f960290](https://github.com/IceRhymers/databricks-claude/commit/f96029008e9fe1132a0449715b150b205928d80a))


### Bug Fixes

* correct YAML syntax in release.yml (missing newline before update-homebrew job) ([#49](https://github.com/IceRhymers/databricks-claude/issues/49)) ([412c01e](https://github.com/IceRhymers/databricks-claude/commit/412c01eff6e1565003730f2d28fe8472db174d23))

## [0.8.1](https://github.com/IceRhymers/databricks-claude/compare/v0.8.0...v0.8.1) (2026-04-07)


### Bug Fixes

* use platform-specific builds for syscall.Flock and syscall.Umask ([16af70b](https://github.com/IceRhymers/databricks-claude/commit/16af70bf566b0a585fa09c38ab41ad0942e8ff7a))

## [0.8.0](https://github.com/IceRhymers/databricks-claude/compare/v0.7.2...v0.8.0) (2026-04-07)


### Features

* add --headless flag for proxy-only startup ([#43](https://github.com/IceRhymers/databricks-claude/issues/43)) ([3141f92](https://github.com/IceRhymers/databricks-claude/commit/3141f926f95bf89b8faed083433512cba2de09d3))

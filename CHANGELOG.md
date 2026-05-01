# Changelog

## [0.15.0](https://github.com/IceRhymers/databricks-claude/compare/v0.14.0...v0.15.0) (2026-05-01)


### Features

* **desktop:** add generate-trust-profile subcommand and --for-pkg flag ([fce4dee](https://github.com/IceRhymers/databricks-claude/commit/fce4deeca59d628cee2c45d9e6c9f9be3ec2cc36))
* per-signal OTel control and --otel-traces for Claude Code trace export ([#102](https://github.com/IceRhymers/databricks-claude/issues/102)) ([f0f1455](https://github.com/IceRhymers/databricks-claude/commit/f0f145525b5a2332db8cf5f79a1bd7066f9d56f3))


### Bug Fixes

* add -legacy flag to openssl pkcs12 export for macOS compat ([6250891](https://github.com/IceRhymers/databricks-claude/commit/62508911a7d4a9b6dae8cd348e9f36c00dd2b1e2))
* add -legacy flag to openssl pkcs12 export for macOS compatibility ([0c8ee40](https://github.com/IceRhymers/databricks-claude/commit/0c8ee4024fc696ac70bceed851665a73763f3442))
* add Apple installer OID to cert so productsign accepts it ([8893d84](https://github.com/IceRhymers/databricks-claude/commit/8893d848ca5d80dc736cba746dc8184d70f244f8))
* add installer OID to cert and guard productsign failure ([691c576](https://github.com/IceRhymers/databricks-claude/commit/691c57687272d6de49b61cd6eacb422a0c03734a))
* add keyUsage=critical,digitalSignature to generate-signing-cert ([dc60c9d](https://github.com/IceRhymers/databricks-claude/commit/dc60c9d1b9dcc881d87b6b6b79431eda0939bdd8))
* add keyUsage=digitalSignature to cert — required for macOS codesign ([8ce2130](https://github.com/IceRhymers/databricks-claude/commit/8ce2130936d343dd2653c22478aeb06615f25609))
* allow pkgutil --check-signature to exit non-zero for unsigned pkg ([dd4b7e2](https://github.com/IceRhymers/databricks-claude/commit/dd4b7e2ad387d2d8f5aad936dd68c429c5b6ffd4))
* allow pkgutil --check-signature to exit non-zero for unsigned pkg ([c3e80d3](https://github.com/IceRhymers/databricks-claude/commit/c3e80d3ef22307c12fb4603de3685976c8c29b1f))
* expand pkg archive before checking payload for helper path ([7fd592c](https://github.com/IceRhymers/databricks-claude/commit/7fd592cb23f26dd8125ff29e761d9c2f2089c527))
* explicitly unlock + list keychain before codesign; add identity diagnostic ([9af5fea](https://github.com/IceRhymers/databricks-claude/commit/9af5fea5452daf2da1b3166e8a88a787c16d1a9c))
* explicitly unlock + list keychain before codesign; add identity diagnostic ([ff6e57f](https://github.com/IceRhymers/databricks-claude/commit/ff6e57fa7b9d507d495c4331681e6f6926dfead2))
* include credential-helper symlink in pkg payload ([a5e6eb5](https://github.com/IceRhymers/databricks-claude/commit/a5e6eb5c5d991c6dba8f26554d9577a299b59b8d))
* include helper symlink in pkg payload; fix assertion to expand archive ([eb3dc78](https://github.com/IceRhymers/databricks-claude/commit/eb3dc788e42d0eaa0f3805fb0fefd09ec2acc60b))
* remove productsign — self-signed certs can't satisfy installer policy ([00e623c](https://github.com/IceRhymers/databricks-claude/commit/00e623c4a6064d2cfa09ca6ca946a1b076cb142d))
* remove trust step, assert pkg is unsigned ([227143f](https://github.com/IceRhymers/databricks-claude/commit/227143f13902da8ea9067ba30346e0c4337e7261))
* ship unsigned .pkg — productsign requires Apple-issued installer cert ([5e94ea2](https://github.com/IceRhymers/databricks-claude/commit/5e94ea20ff45dd5dff89d87332bc7822a41cf261))
* trust self-signed cert as system root so productsign accepts it ([3ae0f9c](https://github.com/IceRhymers/databricks-claude/commit/3ae0f9cb98b7238807c2cbe299e47e15d6ba7011))
* trust self-signed cert as system root so productsign accepts it ([72ecb6c](https://github.com/IceRhymers/databricks-claude/commit/72ecb6cabdef7dc646ca92dc1357f118338125f7))
* use action step output for keychain password in set-key-partition-list ([5b514bd](https://github.com/IceRhymers/databricks-claude/commit/5b514bd708c7dc0202f7e060a81051ef8085b9c1))
* use action step output for keychain password in set-key-partition-list ([f81d961](https://github.com/IceRhymers/databricks-claude/commit/f81d9611fcb799de397a2bfaeb2157dfeef38903))
* use APPLE_INTERNAL_SIGNING_IDENTITY secret in pkg signature assert ([f14f42b](https://github.com/IceRhymers/databricks-claude/commit/f14f42bfe16f02bb06de6d8d2e7ee2af8e0077e1))
* use APPLE_INTERNAL_SIGNING_IDENTITY secret in pkg signature assert ([ac0a3f5](https://github.com/IceRhymers/databricks-claude/commit/ac0a3f59df80b2ff891030615dc1adbe8eb68f6b))

## [0.14.0](https://github.com/IceRhymers/databricks-claude/compare/v0.13.0...v0.14.0) (2026-04-28)


### Features

* --install-hooks performs first-run env setup; hook–desktop coexist ([da43ab9](https://github.com/IceRhymers/databricks-claude/commit/da43ab944e5cf4384061b6f08d17a7b59e372496))
* --install-hooks performs first-run env setup; hook–desktop coexist ([abf9871](https://github.com/IceRhymers/databricks-claude/commit/abf9871e585ac730e15a88d5e4e8165bd61e893b))
* emit databricks-claude-desktop.json for Claude Desktop dev mode ([313ae51](https://github.com/IceRhymers/databricks-claude/commit/313ae51ba0cbdbb3d32affc5d3ac9e13281f6f95))
* emit databricks-claude-desktop.json for Claude Desktop dev mode ([9c1af01](https://github.com/IceRhymers/databricks-claude/commit/9c1af01934ac001512b648fc2757b4548248ec1a))


### Bug Fixes

* update README to clarify Claude Desktop's inference management ([cdb61d0](https://github.com/IceRhymers/databricks-claude/commit/cdb61d0a0573d1d3160b8afdb603221ca44d1e57))

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

# Changelog

## [0.18.0](https://github.com/IceRhymers/databricks-claude/compare/v0.17.0...v0.18.0) (2026-05-06)


### Features

* --credential-helper and --generate-desktop-config for Claude Desktop ([12ef44c](https://github.com/IceRhymers/databricks-claude/commit/12ef44cea5cd4577ce6c800cb3ffba6ad3f17b4f))
* --install-hooks performs first-run env setup; hook–desktop coexist ([da43ab9](https://github.com/IceRhymers/databricks-claude/commit/da43ab944e5cf4384061b6f08d17a7b59e372496))
* --install-hooks performs first-run env setup; hook–desktop coexist ([abf9871](https://github.com/IceRhymers/databricks-claude/commit/abf9871e585ac730e15a88d5e4e8165bd61e893b))
* add --credential-helper and --generate-desktop-config for Claude Desktop ([00e05c1](https://github.com/IceRhymers/databricks-claude/commit/00e05c1c85c61235129b39136cd3140664c21b8e)), closes [#92](https://github.com/IceRhymers/databricks-claude/issues/92)
* add --headless flag for proxy-only startup ([#43](https://github.com/IceRhymers/databricks-claude/issues/43)) ([3141f92](https://github.com/IceRhymers/databricks-claude/commit/3141f926f95bf89b8faed083433512cba2de09d3))
* add --help UX with claude passthrough and --print-env debug flag ([c483b5b](https://github.com/IceRhymers/databricks-claude/commit/c483b5ba936f3890d1a37e7bd4f31627dd4c8e7f))
* add --install-hooks, --uninstall-hooks, and proxy lifecycle hooks ([#63](https://github.com/IceRhymers/databricks-claude/issues/63)) ([1e2b84f](https://github.com/IceRhymers/databricks-claude/commit/1e2b84f2f008dab16bbe4eeee673f9a72a29a46d))
* add --no-otel flag and wire otelKeysPersistent to SettingsManager ([c328165](https://github.com/IceRhymers/databricks-claude/commit/c32816594003e1cd866150745143bf54d8ba80e8))
* add otelKeysPersistent to SettingsManager for selective restore ([2680490](https://github.com/IceRhymers/databricks-claude/commit/2680490546121293ff898f1612f214bd24a73199))
* add pkg/updater — GitHub release checker with cache and numeric semver comparison ([#75](https://github.com/IceRhymers/databricks-claude/issues/75)) ([32e58cf](https://github.com/IceRhymers/databricks-claude/commit/32e58cf52fa7837f59283d4c921d9472f855dd63))
* add POST /shutdown endpoint and idle timeout for headless lifecycle management ([#61](https://github.com/IceRhymers/databricks-claude/issues/61)) ([fe184d1](https://github.com/IceRhymers/databricks-claude/commit/fe184d142019c0348793dd8a1538dd166c987557))
* add shell tab completions (bash/zsh/fish) ([#69](https://github.com/IceRhymers/databricks-claude/issues/69)) ([7b011e4](https://github.com/IceRhymers/databricks-claude/commit/7b011e471f05a320d1407372248ebe0d8adbdad7))
* add update subcommand and startup update check ([#77](https://github.com/IceRhymers/databricks-claude/issues/77)) ([0e9b525](https://github.com/IceRhymers/databricks-claude/commit/0e9b525b0da0cff2246daa12d685b4352192af57))
* **desktop:** add generate-trust-profile subcommand and --for-pkg flag ([fce4dee](https://github.com/IceRhymers/databricks-claude/commit/fce4deeca59d628cee2c45d9e6c9f9be3ec2cc36))
* dispatch Homebrew formula update on release ([#48](https://github.com/IceRhymers/databricks-claude/issues/48)) ([f960290](https://github.com/IceRhymers/databricks-claude/commit/f96029008e9fe1132a0449715b150b205928d80a))
* document release-please conventional commit requirements in AGENTS.md ([bd3e865](https://github.com/IceRhymers/databricks-claude/commit/bd3e865162f291a6f07549041306356547425da9))
* emit databricks-claude-desktop.json for Claude Desktop dev mode ([313ae51](https://github.com/IceRhymers/databricks-claude/commit/313ae51ba0cbdbb3d32affc5d3ac9e13281f6f95))
* emit databricks-claude-desktop.json for Claude Desktop dev mode ([9c1af01](https://github.com/IceRhymers/databricks-claude/commit/9c1af01934ac001512b648fc2757b4548248ec1a))
* fixed port session management (closes [#32](https://github.com/IceRhymers/databricks-claude/issues/32)) ([#33](https://github.com/IceRhymers/databricks-claude/issues/33)) ([69fd504](https://github.com/IceRhymers/databricks-claude/commit/69fd504f7c837f4b3a850d9c72a84f6eac51c23a))
* independent OTEL metrics and logs table configuration ([536f33f](https://github.com/IceRhymers/databricks-claude/commit/536f33f38fd5121f25b2ab622478208321a2c423))
* initial extraction of databricks-claude from claude-marketplace-builder ([f17133e](https://github.com/IceRhymers/databricks-claude/commit/f17133e96905895afaedbaa18d5fb2bb656d006c))
* make OTEL config persistent across sessions ([58ec6f4](https://github.com/IceRhymers/databricks-claude/commit/58ec6f4354425fb707070ad7f3a89ab730fe0718))
* per-signal OTel control and --otel-traces for Claude Code trace export ([#102](https://github.com/IceRhymers/databricks-claude/issues/102)) ([f0f1455](https://github.com/IceRhymers/databricks-claude/commit/f0f145525b5a2332db8cf5f79a1bd7066f9d56f3))
* persist Databricks CLI profile across sessions ([8325765](https://github.com/IceRhymers/databricks-claude/commit/8325765d042dd8f55fadfc82e83be549c9e36306))
* persist Databricks CLI profile across sessions ([0e7df4f](https://github.com/IceRhymers/databricks-claude/commit/0e7df4f20b7fbba078ff353baa72558c348d2259)), closes [#17](https://github.com/IceRhymers/databricks-claude/issues/17)
* **pkg/authcheck:** add shared auth detection, adopt in claude main ([#22](https://github.com/IceRhymers/databricks-claude/issues/22)) ([6efdf7e](https://github.com/IceRhymers/databricks-claude/commit/6efdf7edc035baa6e130d3cda58f579e93f1e626)), closes [#21](https://github.com/IceRhymers/databricks-claude/issues/21)
* **pkg/proxy:** add WebSocket support and make UCMetricsTable optional ([b407420](https://github.com/IceRhymers/databricks-claude/commit/b4074203e3ed9081ad0b542e37d31bc6f3523a03)), closes [#19](https://github.com/IceRhymers/databricks-claude/issues/19)
* preserve user-configured model names in FullSetup ([74f53aa](https://github.com/IceRhymers/databricks-claude/commit/74f53aab175b68dc93d0f0fa2ee945dbe5057ff5)), closes [#11](https://github.com/IceRhymers/databricks-claude/issues/11)
* **proxy:** optional API key auth and TLS listener (closes [#26](https://github.com/IceRhymers/databricks-claude/issues/26), closes [#27](https://github.com/IceRhymers/databricks-claude/issues/27)) ([#31](https://github.com/IceRhymers/databricks-claude/issues/31)) ([5754cda](https://github.com/IceRhymers/databricks-claude/commit/5754cdaadddf9bda8a9ae37a0af39d90a0fc41ef))
* require conventional commit prefix in agent instructions ([77fc20e](https://github.com/IceRhymers/databricks-claude/commit/77fc20ea91af17e57686e9d3d4125fea3f85c10f))
* require conventional commit prefix in agent instructions ([a6db721](https://github.com/IceRhymers/databricks-claude/commit/a6db72118d8979bf5122004ba7287318059cff69))


### Bug Fixes

* accept --shell=bash form from Homebrew generate_completions_from_executable ([#71](https://github.com/IceRhymers/databricks-claude/issues/71)) ([4ec2a08](https://github.com/IceRhymers/databricks-claude/commit/4ec2a082879d483b212f3bcf7e022535dc303c62))
* add -legacy flag to openssl pkcs12 export for macOS compat ([6250891](https://github.com/IceRhymers/databricks-claude/commit/62508911a7d4a9b6dae8cd348e9f36c00dd2b1e2))
* add -legacy flag to openssl pkcs12 export for macOS compatibility ([0c8ee40](https://github.com/IceRhymers/databricks-claude/commit/0c8ee4024fc696ac70bceed851665a73763f3442))
* add Apple installer OID to cert so productsign accepts it ([8893d84](https://github.com/IceRhymers/databricks-claude/commit/8893d848ca5d80dc736cba746dc8184d70f244f8))
* add installer OID to cert and guard productsign failure ([691c576](https://github.com/IceRhymers/databricks-claude/commit/691c57687272d6de49b61cd6eacb422a0c03734a))
* add keyUsage=critical,digitalSignature to generate-signing-cert ([dc60c9d](https://github.com/IceRhymers/databricks-claude/commit/dc60c9d1b9dcc881d87b6b6b79431eda0939bdd8))
* add keyUsage=digitalSignature to cert — required for macOS codesign ([8ce2130](https://github.com/IceRhymers/databricks-claude/commit/8ce2130936d343dd2653c22478aeb06615f25609))
* add missing sanitize patterns and reject bare integers for --idle-timeout ([ec2e95b](https://github.com/IceRhymers/databricks-claude/commit/ec2e95be100985df4d1a01bfa7bde1752bf2e183))
* address PR [#93](https://github.com/IceRhymers/databricks-claude/issues/93) review — generate both artifacts, MDM CLI pinning, README docs ([ba7c7b3](https://github.com/IceRhymers/databricks-claude/commit/ba7c7b386bf662d656a15058ec686a066aae9c83))
* allow pkgutil --check-signature to exit non-zero for unsigned pkg ([dd4b7e2](https://github.com/IceRhymers/databricks-claude/commit/dd4b7e2ad387d2d8f5aad936dd68c429c5b6ffd4))
* allow pkgutil --check-signature to exit non-zero for unsigned pkg ([c3e80d3](https://github.com/IceRhymers/databricks-claude/commit/c3e80d3ef22307c12fb4603de3685976c8c29b1f))
* broaden sanitize patterns to cover dapi tokens without hyphen and X-Databricks-Authorization ([7cccfd8](https://github.com/IceRhymers/databricks-claude/commit/7cccfd8f9d4a9d738d3b9e9bd7e3662c9ccecae1))
* call Restore() explicitly before os.Exit instead of defer ([67256a3](https://github.com/IceRhymers/databricks-claude/commit/67256a30f24d719b0fad267a9b40b9b4044b71e4))
* ci trigger on master not main ([8aea836](https://github.com/IceRhymers/databricks-claude/commit/8aea83660bb70b0df97732f24161307d7e19bd17))
* ci trigger on master not main ([fd54481](https://github.com/IceRhymers/databricks-claude/commit/fd54481993ab211e8a76780c5c8e11a4313cf2ba))
* complete OTEL setup — add logs support, differentiate metrics/logs tables, fix --print-env ([8ce428c](https://github.com/IceRhymers/databricks-claude/commit/8ce428c9ae8831d11f67f5d723b6a78aea5515e6)), closes [#7](https://github.com/IceRhymers/databricks-claude/issues/7)
* concurrent session management — refcount release and proxy takeover ([#41](https://github.com/IceRhymers/databricks-claude/issues/41)) ([e933fec](https://github.com/IceRhymers/databricks-claude/commit/e933fec4541408a28574178dc1f2201251fbb119))
* concurrent session safety — flock + session registry ([#9](https://github.com/IceRhymers/databricks-claude/issues/9)) ([2705b49](https://github.com/IceRhymers/databricks-claude/commit/2705b49535bce6564c848b5610d7d7701da76c9d))
* correct YAML syntax in release.yml (missing newline before update-homebrew job) ([#49](https://github.com/IceRhymers/databricks-claude/issues/49)) ([412c01e](https://github.com/IceRhymers/databricks-claude/commit/412c01eff6e1565003730f2d28fe8472db174d23))
* expand pkg archive before checking payload for helper path ([7fd592c](https://github.com/IceRhymers/databricks-claude/commit/7fd592cb23f26dd8125ff29e761d9c2f2089c527))
* explicitly unlock + list keychain before codesign; add identity diagnostic ([9af5fea](https://github.com/IceRhymers/databricks-claude/commit/9af5fea5452daf2da1b3166e8a88a787c16d1a9c))
* explicitly unlock + list keychain before codesign; add identity diagnostic ([ff6e57f](https://github.com/IceRhymers/databricks-claude/commit/ff6e57fa7b9d507d495c4331681e6f6926dfead2))
* include credential-helper symlink in pkg payload ([a5e6eb5](https://github.com/IceRhymers/databricks-claude/commit/a5e6eb5c5d991c6dba8f26554d9577a299b59b8d))
* include helper symlink in pkg payload; fix assertion to expand archive ([eb3dc78](https://github.com/IceRhymers/databricks-claude/commit/eb3dc788e42d0eaa0f3805fb0fefd09ec2acc60b))
* make credential helper work from Claude Desktop's GUI subprocess ([c58b3f0](https://github.com/IceRhymers/databricks-claude/commit/c58b3f02d86e076b64d96d13b3191efbd30445c3))
* normalize trailing newline in main.go ([#52](https://github.com/IceRhymers/databricks-claude/issues/52)) ([f2ae5cf](https://github.com/IceRhymers/databricks-claude/commit/f2ae5cfa1da9ab851388fdeb6b0ad5e24b9962d4))
* OTEL proxy error logging and stale endpoint cleanup ([8355191](https://github.com/IceRhymers/databricks-claude/commit/835519144573f777edff3d29c8c22093d3f48ef2))
* properly wrap lifecycle on health watcher takeover ([17fb740](https://github.com/IceRhymers/databricks-claude/commit/17fb740b2141aef664d04ca04838ec92e2d605fa))
* remove DATABRICKS_HOST and DATABRICKS_CONFIG_PROFILE from settings.json env ([#67](https://github.com/IceRhymers/databricks-claude/issues/67)) ([22649e7](https://github.com/IceRhymers/databricks-claude/commit/22649e7acfc2545803f844a99c85399da5f0689a))
* remove env var from profile resolution — state file always wins ([#35](https://github.com/IceRhymers/databricks-claude/issues/35)) ([e26f7ca](https://github.com/IceRhymers/databricks-claude/commit/e26f7cacd68667fbcfb6d179c1a7ab3ccfd878eb))
* remove productsign — self-signed certs can't satisfy installer policy ([00e623c](https://github.com/IceRhymers/databricks-claude/commit/00e623c4a6064d2cfa09ca6ca946a1b076cb142d))
* remove stale proxy entries from settings.json on startup ([fd7fc35](https://github.com/IceRhymers/databricks-claude/commit/fd7fc3582db6f5cea133cf01649129b4a641fb02))
* remove stale proxy entries from settings.json on startup ([8a1856b](https://github.com/IceRhymers/databricks-claude/commit/8a1856bdf2be3def5b2b4b40bc144f4f82ae49c5))
* remove trust step, assert pkg is unsigned ([227143f](https://github.com/IceRhymers/databricks-claude/commit/227143f13902da8ea9067ba30346e0c4337e7261))
* rename --verbose to stderr output, add --log-file for file redirect ([0253a1c](https://github.com/IceRhymers/databricks-claude/commit/0253a1c1595b10b9425144108e4f1b295349620a)), closes [#3](https://github.com/IceRhymers/databricks-claude/issues/3)
* restore isExecutableFile shim in root package for desktop_config.go ([d1effbb](https://github.com/IceRhymers/databricks-claude/commit/d1effbb0e388b848579f3cf0a81456799bf402b1))
* return errors from NewServer and Ensure instead of log.Fatalf; resolve CLI path in authcheck ([eeaa65e](https://github.com/IceRhymers/databricks-claude/commit/eeaa65ec3ce55b032ca9fd6e5643c13b9cf4cd6b))
* scheme-aware health check and TLS arg forwarding in headless.Ensure ([#84](https://github.com/IceRhymers/databricks-claude/issues/84)) ([fd3b0aa](https://github.com/IceRhymers/databricks-claude/commit/fd3b0aaf0c13100abd8000c11348846444bf62da))
* ship unsigned .pkg — productsign requires Apple-issued installer cert ([5e94ea2](https://github.com/IceRhymers/databricks-claude/commit/5e94ea20ff45dd5dff89d87332bc7822a41cf261))
* skip Authorization header when token empty; fix WebSocket goroutine leak ([b3bf297](https://github.com/IceRhymers/databricks-claude/commit/b3bf2971d403f565194c1ae33c798362a5892657))
* treat EPERM as alive in isProcessAlive — fixes CI failure in containers where Kill(1,0) returns EPERM ([5615bce](https://github.com/IceRhymers/databricks-claude/commit/5615bce5cc91b4c94e587ec372ac3dc1e26ab21c))
* treat missing settings.json as empty document ([#81](https://github.com/IceRhymers/databricks-claude/issues/81)) ([e957c4f](https://github.com/IceRhymers/databricks-claude/commit/e957c4f0db558e797db1609878c111dacb2919c1))
* trust self-signed cert as system root so productsign accepts it ([3ae0f9c](https://github.com/IceRhymers/databricks-claude/commit/3ae0f9cb98b7238807c2cbe299e47e15d6ba7011))
* trust self-signed cert as system root so productsign accepts it ([72ecb6c](https://github.com/IceRhymers/databricks-claude/commit/72ecb6cabdef7dc646ca92dc1357f118338125f7))
* update README to clarify Claude Desktop's inference management ([cdb61d0](https://github.com/IceRhymers/databricks-claude/commit/cdb61d0a0573d1d3160b8afdb603221ca44d1e57))
* use action step output for keychain password in set-key-partition-list ([5b514bd](https://github.com/IceRhymers/databricks-claude/commit/5b514bd708c7dc0202f7e060a81051ef8085b9c1))
* use action step output for keychain password in set-key-partition-list ([f81d961](https://github.com/IceRhymers/databricks-claude/commit/f81d9611fcb799de397a2bfaeb2157dfeef38903))
* use APPLE_INTERNAL_SIGNING_IDENTITY secret in pkg signature assert ([f14f42b](https://github.com/IceRhymers/databricks-claude/commit/f14f42bfe16f02bb06de6d8d2e7ee2af8e0077e1))
* use APPLE_INTERNAL_SIGNING_IDENTITY secret in pkg signature assert ([ac0a3f5](https://github.com/IceRhymers/databricks-claude/commit/ac0a3f59df80b2ff891030615dc1adbe8eb68f6b))
* use os.CreateTemp for unique tmp filenames in settings writers ([a2d97a0](https://github.com/IceRhymers/databricks-claude/commit/a2d97a0b993f21b841346416acf7f2518b320c00))
* use platform-specific builds for syscall.Flock and syscall.Umask ([16af70b](https://github.com/IceRhymers/databricks-claude/commit/16af70bf566b0a585fa09c38ab41ad0942e8ff7a))

## [0.17.0](https://github.com/IceRhymers/databricks-claude/compare/v0.16.0...v0.17.0) (2026-05-06)


### Refactors

* replace `parseArgs` 31-value tuple with `Args` struct — all flag values now accessed as `a.FieldName`; callers updated throughout ([6843844](https://github.com/IceRhymers/databricks-claude/commit/6843844f1c9d8c28e69e2b70b3ee78c15f49e5b))


### Bug Fixes

* remove stale proxy entries from settings.json on startup — probes existing `127.0.0.1:<port>` entries with a 200ms TCP dial and removes dead ones before writing the new URL ([8a1856b](https://github.com/IceRhymers/databricks-claude/commit/8a1856bdf2be3def5b2b4b40bc144f4f82ae49c5))
* `pkg/proxy.NewServer` and `pkg/headless.Ensure` now return errors instead of calling `log.Fatalf`; `authcheck` resolves CLI binary via fallback dirs (fixes silent false-negative on GUI-launched sessions) ([eeaa65e](https://github.com/IceRhymers/databricks-claude/commit/eeaa65ec3ce55b032ca9fd6e5643c13b9cf4cd6b))
* add missing sanitize patterns: case-insensitive Bearer, Basic auth, `access_token` JSON field, `DATABRICKS_TOKEN` env-var; `--idle-timeout` now rejects bare integers with a clear error ([ec2e95b](https://github.com/IceRhymers/databricks-claude/commit/ec2e95be100985df4d1a01bfa7bde1752bf2e183))
* use `os.CreateTemp` for unique tmp filenames in settings writers — eliminates concurrent-write corruption when two processes start simultaneously ([a2d97a0](https://github.com/IceRhymers/databricks-claude/commit/a2d97a0b993f21b841346416acf7f2518b320c00))
* broaden sanitize patterns to cover `dapi` tokens without hyphen and `X-Databricks-Authorization` header ([7cccfd8](https://github.com/IceRhymers/databricks-claude/commit/7cccfd8f9d4a9d738d3b9e9bd7e3662c9ccecae1))
* skip `Authorization` header injection when token is empty; fix WebSocket goroutine leak (single `<-done` replaced with explicit conn close + second receive) ([b3bf297](https://github.com/IceRhymers/databricks-claude/commit/b3bf2971d403f565194c1ae33c798362a5892657))
* properly wrap lifecycle handler on health-watcher takeover so `/shutdown` triggers clean shutdown after promotion ([17fb740](https://github.com/IceRhymers/databricks-claude/commit/17fb740b2141aef664d04ca04838ec92e2d605fa))


### Improvements

* replace hand-rolled slice-bounds prefix checks with `strings.HasPrefix`; add `https://` variant for localhost suppression; fix orphan comment in `pkg/refcount` ([219a29b](https://github.com/IceRhymers/databricks-claude/commit/219a29b))
* align README with current CLI flags and `--print-env` output ([46a4d9b](https://github.com/IceRhymers/databricks-claude/commit/46a4d9b))

## [0.16.0](https://github.com/IceRhymers/databricks-claude/compare/v0.15.0...v0.16.0) (2026-05-04)


### Features

* simplify `ConstructGatewayURL`: use host-relative AI Gateway path (`{host}/ai-gateway/anthropic`), removing SCIM workspace-ID lookup, token parameter, and fallback ([#116](https://github.com/IceRhymers/databricks-claude/issues/116)) ([8fa432a](https://github.com/IceRhymers/databricks-claude/commit/8fa432aca093239fbcc0c31c772d5b36819e42ba))
* require conventional commit prefix in agent instructions ([a6db721](https://github.com/IceRhymers/databricks-claude/commit/a6db72118d8979bf5122004ba7287318059cff69))

## [0.15.0](https://github.com/IceRhymers/databricks-claude/compare/v0.14.0...v0.15.0) (2026-05-01)


### Features

* **otel:** per-signal export control; `--otel-traces` exports Claude Code traces independently of metrics/logs ([#102](https://github.com/IceRhymers/databricks-claude/issues/102)) ([f0f1455](https://github.com/IceRhymers/databricks-claude/commit/f0f145525b5a2332db8cf5f79a1bd7066f9d56f3))
* **desktop:** `generate-trust-profile` subcommand with `--for-pkg` flag for pkg-scoped trust profiles ([fce4dee](https://github.com/IceRhymers/databricks-claude/commit/fce4deeca59d628cee2c45d9e6c9f9be3ec2cc36))


### Bug Fixes

* macOS `.pkg` ships unsigned — `productsign` requires an Apple-issued installer cert; build pipeline now asserts unsigned rather than attempting signing ([5e94ea2](https://github.com/IceRhymers/databricks-claude/commit/5e94ea20ff45dd5dff89d87332bc7822a41cf261))
* credential-helper symlink included in pkg payload; archive expanded before payload assertion ([eb3dc78](https://github.com/IceRhymers/databricks-claude/commit/eb3dc788e42d0eaa0f3805fb0fefd09ec2acc60b))
* explicitly unlock keychain and list identities before `codesign` ([9af5fea](https://github.com/IceRhymers/databricks-claude/commit/9af5fea5452daf2da1b3166e8a88a787c16d1a9c))

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

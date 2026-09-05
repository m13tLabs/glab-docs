
## [0.2.1](https://github.com/m13tLabs/renovate-config/compare/v0.2.0...v0.2.1) (2026-09-05)


## [0.2.0](https://github.com/m13tLabs/renovate-config/compare/v0.1.0...v0.2.0) (2026-09-05)

### Dependency Updates

* **deps:** Update docker/build-push-action action to v6.19.2 ([81f83c5](https://github.com/m13tLabs/renovate-config/commit/81f83c5b226111d78c45d7622e9db47622eb87f5))

* **deps:** Update docker/metadata-action action to v5.10.0 ([29dd3e6](https://github.com/m13tLabs/renovate-config/commit/29dd3e6edf51bd3ba1a8443127558b9d76a76f4d))

* **deps:** Update golang docker tag to v1.27 ([0851a02](https://github.com/m13tLabs/renovate-config/commit/0851a02d5da59091255456e002b970ec74bb8b7f))

* **deps:** Update docker/build-push-action action to v7 ([b84da5e](https://github.com/m13tLabs/renovate-config/commit/b84da5e5dbfe081e5e05e26386101fc470a2ca94))

* **deps:** Update docker/metadata-action action to v6 ([e778c0b](https://github.com/m13tLabs/renovate-config/commit/e778c0be72df8cc7873a5b3c189aaa1d6232fafc))



### Documentation

* Update README ([3713730](https://github.com/m13tLabs/renovate-config/commit/3713730dd730cdc40f5ffff99c70c7cdb4d542ea))



### Features

* Document pipeline jobs ([1881556](https://github.com/m13tLabs/renovate-config/commit/18815562d9a98cd67c6b3d2b371a6d95e96db852))

* **Gitlab:** Update image tag in gitlab component ([ed7db10](https://github.com/m13tLabs/renovate-config/commit/ed7db10e4527ed8ac5cd31c64335c47bfaa55bec))




# [0.1.0](https://github.com/m13tLabs/glab-docs/compare/71d462776e61752769792f70a857211e4484d150...v0.1.0) (2026-09-04)


* feat!: Adds the capability to provide comments without the path to the documented field fixes #58 ([4634ab3](https://github.com/m13tLabs/glab-docs/commit/4634ab3968f34995195e72cc3fd8f285eba52ca2)), closes [#58](https://github.com/m13tLabs/glab-docs/issues/58)


### Bug Fixes

* Adds check for tag format to release script, also makes release script error on improper tags ([5a5d142](https://github.com/m13tLabs/glab-docs/commit/5a5d142c452f9e2ae45d7923c4f7c7bb7be53f3c))
* change error to err to not conflict with builtin interface ([5a87421](https://github.com/m13tLabs/glab-docs/commit/5a87421808b2728aa8590046a40900bb395ff99e))
* change rendering of <nil> in markdown to render correctly ([58d915d](https://github.com/m13tLabs/glab-docs/commit/58d915d430b672a2e31deeeda254b034b2c3f04c))
* changed to always get the latest version on the helm docs pre commit ([d38ede8](https://github.com/m13tLabs/glab-docs/commit/d38ede8c32812e0ec0799cebf269b0225ae38b37))
* cleans up some code and minor fix to ignore case ([33f87b1](https://github.com/m13tLabs/glab-docs/commit/33f87b1171ed39ecb303352d7d202abd5b0f01b2))
* correct contributing link referent in pr template ([a808d6e](https://github.com/m13tLabs/glab-docs/commit/a808d6e178fa071f490010d8d962ecccd8e83ddc))
* Correct the name of the GitHub repository in the README ([51fe584](https://github.com/m13tLabs/glab-docs/commit/51fe58474f1fcb29c750a8cea1d00348241fd777)), closes [#208](https://github.com/m13tLabs/glab-docs/issues/208)
* **deps:** update module github.com/sirupsen/logrus to v1.10.2 ([885598c](https://github.com/m13tLabs/glab-docs/commit/885598c0c838ce880098da1dc04c4bc48ece7431))
* **deps:** update module github.com/sirupsen/logrus to v1.9.4 ([066f8a6](https://github.com/m13tLabs/glab-docs/commit/066f8a64ad72022479b08ced5c4663d98ae78809))
* **deps:** update module helm.sh/helm/v3 to v4 ([ca6df5e](https://github.com/m13tLabs/glab-docs/commit/ca6df5ef808e9231cfc51a2fe969bc725e82b63d))
* **docker:** drop unpinned apk install to satisfy hadolint ([422fcc8](https://github.com/m13tLabs/glab-docs/commit/422fcc83314850e0482db5ec37df213b560e6aa8))
* Doesn't backtick-quote custom default values, back to the way originally implemented ([a792182](https://github.com/m13tLabs/glab-docs/commit/a792182ed32f614412e4112cc5075e5cc2b8d6ae))
* escapes dashes in version badges so complicated versions work fixes [#56](https://github.com/m13tLabs/glab-docs/issues/56) ([4ad6c82](https://github.com/m13tLabs/glab-docs/commit/4ad6c82e6c905c66c4d9d2600b667c048a43b25c))
* fixes [@default](https://github.com/default) for variables without value ([a77f2e1](https://github.com/m13tLabs/glab-docs/commit/a77f2e1ef9e6bae81c85526f07f59fb59b651bb8))
* fixes broken comment parsing on values files with dos line endings ([e65dfab](https://github.com/m13tLabs/glab-docs/commit/e65dfaba10d1f5b9dc85f4de8d8b7892af1fecd9))
* fixes docker image name ([effae03](https://github.com/m13tLabs/glab-docs/commit/effae032ba20cc7a6874e362cbc33e8c88322c46))
* fixes file operations to work when not running from the chart root and fixes several tests ([45f63df](https://github.com/m13tLabs/glab-docs/commit/45f63df3a13c43bdcb30c26a58eeac40fcf87dab))
* Fixes goreleaser action by using new flag ([ff7941b](https://github.com/m13tLabs/glab-docs/commit/ff7941bae8026af923dea250602340ad03d1878d))
* fixes small formatting issue ([ec63edc](https://github.com/m13tLabs/glab-docs/commit/ec63edc314707e8a6bd175043f1536afce580d88))
* fixes tests by calling correct method ([445a511](https://github.com/m13tLabs/glab-docs/commit/445a511bd67e0a828fea32d2339d097006231575))
* generate docs for nil values and include types for them ([71d4627](https://github.com/m13tLabs/glab-docs/commit/71d462776e61752769792f70a857211e4484d150))
* GitHub token for Homebrew tap ([26112f1](https://github.com/m13tLabs/glab-docs/commit/26112f182d9892f32f3355ff936ec0e95c5fb8ed))
* **goreleaser:** adding more os and arch types ([6e0311d](https://github.com/m13tLabs/glab-docs/commit/6e0311db5e2a17fe3d5a262a5645035c6ab73202))
* link to hashbash ([adb421c](https://github.com/m13tLabs/glab-docs/commit/adb421c6971da41a5f8fe1912a39b24301349d53))
* makes description appear even if unrelated comment appears before description comment fixes [#92](https://github.com/m13tLabs/glab-docs/issues/92) ([aaec4ec](https://github.com/m13tLabs/glab-docs/commit/aaec4ec4a172ba5a939621f894cda4522e33d4cd))
* makes it so charts with empty values files still get helm-docs-version footers ([8bb2011](https://github.com/m13tLabs/glab-docs/commit/8bb201129e35f6f3c2852c426f71606133cb187e))
* no double whitespace using the badges ([b3a41ff](https://github.com/m13tLabs/glab-docs/commit/b3a41ffaeaab75aa9f6ca9e9d0a9b67825a46cd3))
* **README:** change the way helm-docs is installed ([eb95c9d](https://github.com/m13tLabs/glab-docs/commit/eb95c9d931a4b34e87993cacceb7ed0c9670ef80))
* remove var env dependency by moving tests ([3f2182a](https://github.com/m13tLabs/glab-docs/commit/3f2182a8c735ee6d83f22530ae37e138164bd6a4))
* removing goreleaser project env var to be able to test locally ([d0e53ef](https://github.com/m13tLabs/glab-docs/commit/d0e53ef9312bed7432f4fa849f7e879ec9e88e85))
* renders <, >, and & characters from default values correctly (fixes [#24](https://github.com/m13tLabs/glab-docs/issues/24)) ([0e8e6fd](https://github.com/m13tLabs/glab-docs/commit/0e8e6fd9fde00941995a94b7c76edbf257eb1bf2))
* small issue in sorting by file location ([3dd414c](https://github.com/m13tLabs/glab-docs/commit/3dd414c052fbf78c2fe5efcd3d1cf0462596e27f))
* Solves [#217](https://github.com/m13tLabs/glab-docs/issues/217) where helm-docs would segfault due in charts with certain comment format ([dd5640c](https://github.com/m13tLabs/glab-docs/commit/dd5640ca11fd2a7a2f77a202b06f680ece880949))
* types on nil values ([93aef0b](https://github.com/m13tLabs/glab-docs/commit/93aef0bae5fb78889f8faf21542bc50df47f93d2))
* update actions ([0d19e63](https://github.com/m13tLabs/glab-docs/commit/0d19e63d853cb85df5e10022e9c9cd98d9a01238))
* update goreleaser and way to get env ([37a3481](https://github.com/m13tLabs/glab-docs/commit/37a3481bdd7588f2855b83334070e6fc04d38515))
* updates name of version var to get a version in the CLI flag ([5847981](https://github.com/m13tLabs/glab-docs/commit/584798160536142172bcfc2194e847df3dc894e9))
* updates signing key so release builds can work again ([40f1dcf](https://github.com/m13tLabs/glab-docs/commit/40f1dcf583f48aeed079a39fcd84570e251c01c3))
* Updating hook config for README.md.gotmpl ([f67f99d](https://github.com/m13tLabs/glab-docs/commit/f67f99de7e3eb6c6928fff432cbc208cf578b046))


### Features

* add full-template example ([feaeece](https://github.com/m13tLabs/glab-docs/commit/feaeece51df20091415721cd597c223a44e0ab93))
* add support for sorting based on presence in file ([de4bf77](https://github.com/m13tLabs/glab-docs/commit/de4bf77711dc08bd18b52df698b2ce4af74e554e))
* add the glab-docs GitLab CI/CD component ([78841e7](https://github.com/m13tLabs/glab-docs/commit/78841e7a0f3b3efa57e0f390df07a6bb79e8fdc0))
* add toYaml and fromYaml to functions map ([d4d0176](https://github.com/m13tLabs/glab-docs/commit/d4d017621e8b846e321ebace2dc99764dfb8cc73))
* add two regex for markdown linting ([8fb6f38](https://github.com/m13tLabs/glab-docs/commit/8fb6f382ebeafae56ea2870f45477153a6cbf030))
* adds a --output-file cli option for specifying the file to which documentation is written (fixes [#28](https://github.com/m13tLabs/glab-docs/issues/28)) ([37501cb](https://github.com/m13tLabs/glab-docs/commit/37501cbf599e7e170b347d4ad25e5331e62eef28))
* adds a new chart with good examples, cleans up README a bit more, and shhhh... fixes a bug ([4acbc68](https://github.com/m13tLabs/glab-docs/commit/4acbc687f798c40befeb5fa19bfb3d5677c20595))
* adds goreleaser configuration to deploy a homebrew tap ([4e79480](https://github.com/m13tLabs/glab-docs/commit/4e79480ae62a6a2ed636e006cbf8c262e5e5d966))
* adds goreleaser to create proper releases, creates packages, adds cobra/viper ([79c45c7](https://github.com/m13tLabs/glab-docs/commit/79c45c7567e0e6e4adf4368847ce1345016f8fa1))
* adds goreport to the readme ([8786369](https://github.com/m13tLabs/glab-docs/commit/87863695f0c684590448e7b58870bb7cce30d8cd))
* adds support for an ignore file to exclude charts from processing ([68966e5](https://github.com/m13tLabs/glab-docs/commit/68966e54629fb1ce64bdb4d0f127fd29721ac834))
* changes files for git hook slightly ([99f737f](https://github.com/m13tLabs/glab-docs/commit/99f737fd424641ed1cfa31a4a5b4398978f2a5e8))
* check for line templates, update README.md ([64ad9ea](https://github.com/m13tLabs/glab-docs/commit/64ad9ea6c0fea3763420d647b5afddf82f530fae))
* deprecation, badges, styling, more options ([77673fb](https://github.com/m13tLabs/glab-docs/commit/77673fb6b5b35166e8980e6877893a53cf1f2c22))
* fixed and updated READMEs ([61d5caf](https://github.com/m13tLabs/glab-docs/commit/61d5caf46d11fef28088eec6bff67912719cb64d))
* improves ignore feature, accepting an ignore file at the root of the repository as well as in the directory ([d6aca61](https://github.com/m13tLabs/glab-docs/commit/d6aca614a7e057fb8c43df6b68c384fd677a33b8))
* Largely expands all features for old comments to new comments, old-style comments effectively deprecated ([d88cca4](https://github.com/m13tLabs/glab-docs/commit/d88cca4131169f33f545d8bacd9261c6bfcc3591))
* make the badge style from shields.io configurable ([fa0ab6f](https://github.com/m13tLabs/glab-docs/commit/fa0ab6f781fe856f2232e356fc10c662bc9c82a7))
* pivot helm-docs fork to document GitLab CI YAML ([ca099d3](https://github.com/m13tLabs/glab-docs/commit/ca099d39cffb7003bb5abf6a2fa50e750347e2f5))
* refactor to use gotemplates to render documentation, adds new example charts ([2a1c431](https://github.com/m13tLabs/glab-docs/commit/2a1c431216c3cac52714f8c922e0bb506f287440))
* update to use go1.22 ([4a68682](https://github.com/m13tLabs/glab-docs/commit/4a686820b465e76fd99f88e9829ec1e7ce94f7fd))
* updates default chart to add a footer to markdown files with the helm-docs version, if set ([670be38](https://github.com/m13tLabs/glab-docs/commit/670be38e68bde3b74b1b417e8c0e5fbddb0c4815))
* updates documentation with homebrew installation instructions ([5d75e92](https://github.com/m13tLabs/glab-docs/commit/5d75e921fe7ccb9935e9317a3ffc967d18793c21))
* updates links to source in chart files ([a16a110](https://github.com/m13tLabs/glab-docs/commit/a16a1106bd9c3383304421fbc1db975cb4e25aa2))
* updates signature public key for action ([1af69cd](https://github.com/m13tLabs/glab-docs/commit/1af69cd8c859f5d0e41ced7985c5134b41506083))
* updates to add chart search path flag and to search for template files differently based on how they're presented fixes [#47](https://github.com/m13tLabs/glab-docs/issues/47) ([02f4425](https://github.com/m13tLabs/glab-docs/commit/02f442526caa545b1c3c0f5520bab90a8df3db8e))
* updates values table generation allowing for non-empty lists/maps to be documented ([3ccb4ed](https://github.com/m13tLabs/glab-docs/commit/3ccb4ed1682e8c0c6db04c7a294e95f72060e0b7))


### Reverts

* Revert "Add angle brackets around urls in requirementsTable" ([02caaaf](https://github.com/m13tLabs/glab-docs/commit/02caaaf59223e00c2283d82499831f1e874f9c9b))


### BREAKING CHANGES

* Completely new behavior with non-special comments

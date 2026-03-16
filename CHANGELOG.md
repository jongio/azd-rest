## [0.4.6] - 2026-03-16

- fix: reorder release steps - registry before cosign/SBOM (0c3b3c0)

## [0.4.5] - 2026-03-16

- fix: correct cosign-installer SHA for v3 (a78908a)
- fix: release workflow - add shell:bash, limit build to ubuntu, fix gosec (9be3706)
- fix: remove non-existent 'build' job from release.yml needs (880df27)
- ci: Bump actions/github-script from 7.1.0 to 8.0.0 (#29) (96eae7d)
- ci: Bump actions/setup-go from 5.6.0 to 6.3.0 (#28) (1456686)
- ci: Bump actions/cache from 4.2.3 to 5.0.3 (#27) (1cec67d)
- ci: Bump actions/checkout from 4.3.1 to 6.0.2 (#26) (d276f89)
- ci: Bump codecov/codecov-action from 4.6.0 to 5.5.2 (#25) (670a86b)
- ci: Bump actions/upload-artifact from 4.6.2 to 7.0.0 (#24) (31f23b1)
- ci: Bump actions/download-artifact from 4.3.0 to 8.0.1 (#23) (1e37d7c)
- ci: Bump anchore/sbom-action from 0.20.0 to 0.23.1 (#22) (39798d9)
- ci: Bump pnpm/action-setup (#21) (a5d051e)
- ci: Bump actions/setup-node from 4.4.0 to 6.3.0 (#20) (6c99b34)
- feat: dispatch-parity quality improvements (#19) (17dfd96)
- chore: update registry for v0.4.4 (9d4da9c)
- chore: replace "Leverage" with "Use" in spec docs (633896e)

## [0.4.4] - 2026-03-12

- chore: update azd-core to v0.5.6 (#17) (2adbcfd)
- ci: optimize GitHub Actions workflows (#16) (5727ec9)
- chore: update registry for v0.4.3 (d8c7907)
- fix: remove hardcoded azd-core version from test (f282e08)

## [0.4.3] - 2026-03-03

- fix: upgrade azd-core to v0.5.5 for resilient credential chain (#14)

## [0.4.2] - 2026-03-03



## [0.4.0] - 2026-03-02

- fix: update build.sh to support multi-platform builds for release (d8923a4)

## [0.3.0] - 2026-03-02



## [0.2.0] - 2026-03-02

- feat: adopt azdext SDK helpers — full extension framework migration (#7) (a4be26e)
- fix: use correct repo creation dates for LinkedIn publish date (04ecfeb)
- fix: add LinkedIn publish date via azd-web-core v2.3.0 (204f620)
- fix: add LinkedIn author meta tag via azd-web-core v2.2.0 (637c631)
- fix: remove duplicate twitter meta tags (handled by shared Layout) (81916aa)
- chore: update azd-web-core to v2.1.0 (Twitter Card tags) (838ee95)
- feat: add social media OG images and consistent titles (036db28)
- feat(web): redesign website with shared azd-web-core design system (#6) (d7d7522)
- Regenerate OG image using Playwright screenshot (eb14ea5)
- Fix OG image: flatten alpha channel to RGB (b2cbdb2)
- Align OG meta tags with azd-exec pattern (591cb66)
- Add OG image and enhance social meta tags (df671b3)
- fix: make release workflow idempotent and handle retries (d4a0b60)
- chore: update registry for v0.1.1 with all platforms (a062ac0)

## [0.1.1] - 2026-02-24

- azd rest - Authenticated Azure REST calls (052b630)

# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-01-15
- Initial project scaffolding aligned with azd-exec (directories, metadata, and tooling configs).

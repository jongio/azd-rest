## [0.5.0] - 2026-07-07

- feat: add csv output format for arrays and value responses (#221) (38b8a3b)
- feat: restrict request hosts with --allow-host (#229) (7c71b5d)
- feat: mask sensitive response fields with --redact (#227) (247b706)
- feat: add --table-columns to choose and order table columns (#226) (1e9df75)
- feat: build JSON request bodies from key=value fields (#225) (8daf4fe)
- feat: add yaml output format (#224) (d283c38)
- feat: add --dump-headers to write response headers to a file (#223) (3c77182)
- feat: add --header-file to load headers from a file (#186) (9ac5e5c)
- Add JMESPath response query flag (#146) (668e286)
- feat: add --client-request-id flag for Azure request correlation (#184) (43a242a)
- deps: upgrade all dependencies to latest (#231) (205912f)
- deps: upgrade all dependencies to latest (#230) (5dd0790)
- feat: surface Azure rate-limit headers with --show-throttle (#208) (a1ce9d7)
- feat: add --form-field for form-urlencoded bodies (#207) (d9400b7)
- feat: add --repeat to benchmark a request with latency stats (#206) (a9e1260)
- feat: add --color for syntax-highlighted JSON output (#205) (7314625)
- feat: add --write-out for curl-style response metadata (#204) (177c7e3)
- Add doctor command to diagnose auth and scope issues (#197) (fdf91a9)
- Add jsonl output format (#196) (1f0fa49)
- Add table output format (#195) (647f366)
- Add graph command for Azure Resource Graph queries (#194) (01b73e3)
- Add whoami command to show the signed-in Azure identity (#193) (569deaf)
- Expose MCP request controls (#147) (fe32e7c)
- feat: add --include flag to show response headers (#169) (#178) (5d1a61b)
- feat: add --silent flag to suppress stderr diagnostics (#171) (#180) (a056d21)
- feat(mcp): add --read-only flag to expose only read tools (#170) (#179) (f160e2a)
- feat: support AZD_REST_* environment variable defaults for flags (#172) (#181) (e62dfca)
- Add scope command to preview detected OAuth scope for a URL (#182) (8879952)
- feat: add --max-time overall request budget (#185) (d0696d6)
- deps: upgrade all dependencies to latest (#209) (ee7c079)
- deps: upgrade all dependencies to latest (#198) (864cc80)
- Add repeatable --url-param for URL query parameters (#183) (56f8952)
- deps: upgrade all dependencies to latest (#187) (9388457)
- deps: update all dependencies to latest (#168) (bb0bea5)
- deps: upgrade all dependencies to latest (#167) (f1cb648)
- deps: upgrade all dependencies to latest (#162) (df988b0)
- deps: update all dependencies to latest (#154) (ea6532d)
- Add api-version helper (#145) (c44bd7a)
- chore: update registry for v0.4.9 (a8e7e0f)

## [0.4.9] - 2026-06-26

- chore(deps): upgrade all dependencies to latest (#143) (70c8655)
- ci: Bump actions/upload-artifact from 7.0.0 to 7.0.1 (#111) (0d153e2)
- ci: Bump codecov/codecov-action from 6.0.0 to 7.0.0 (#126) (7392a1d)
- ci: Bump actions/checkout from 6.0.2 to 7.0.0 (#137) (20bd6b2)
- ci: Bump pnpm/action-setup from 6.0.8 to 6.0.9 (#129) (95d32ec)
- deps: upgrade all dependencies to latest (#138) (b843cec)
- chore: update registry for v0.4.8 (edcdf03)

## [0.4.8] - 2026-06-06

- fix: update cosign to use --bundle format (deprecated --output-signature) (#123) (de4d6f3)
- chore: update registry for v0.4.7 (04de06a)

## [0.4.7] - 2026-06-06

- fix: add subtest to cspell word list (9194d3d)
- fix: add missing cspell words to fix spell check CI (63ed5e9)
- fix: move nolint directive inline to satisfy gofmt (e0b882e)
- fix: suppress tparallel on NoRaceCondition test with explanation (9934ebd)
- fix: add godoc to New*Command stubs, use t.Cleanup in concurrent test (eeded69)
- fix: update CI workflows for go1.26.4 and Node.js 22 (e86ae94)
- fix: bump Go to 1.26.4, upgrade x/net to v0.55.0, refresh pnpm lockfile (1765b86)
- fix: address code review findings - constants, flags, security docs, tests (1de4775)
- refactor: DI, table-driven factories, service layer extraction (8ae4795)
- ci: Bump pnpm/action-setup from 5.0.0 to 6.0.8 (b42062f)
- ci: Bump sigstore/cosign-installer from 4.1.1 to 4.1.2 (5deef3b)
- deps: Bump @types/node from 25.3.0 to 25.9.0 in /web (81ea352)
- deps: Bump astro from 5.17.3 to 6.3.4 in /web (31c83d7)
- ci: Bump actions/setup-node from 6.3.0 to 6.4.0 (3e1672d)
- ci: Bump actions/cache from 5.0.4 to 5.0.5 (73d556e)
- ci: Bump actions/github-script from 8.0.0 to 9.0.0 (445c6d3)
- fix: quality improvements, perf, a11y, and web refactoring (cc41734)
- test: strengthen assertions, add coverage, and integration tests (b47709d)
- ci: workflow improvements, version alignment, and docs fixes (0d78edc)
- chore(web): remove unused autoprefixer, postcss deps and redundant sr-only CSS (33f92d9)
- fix(ci): prevent script injection via expression interpolation in workflows (e45ef91)
- fix(deps): update Go and Node.js dependencies to latest stable versions (2aad79c)
- ci: Bump codecov/codecov-action from 5.5.2 to 6.0.0 (b7824f8)
- ci: Bump actions/setup-go from 6.3.0 to 6.4.0 (90d4003)
- ci: Bump sigstore/cosign-installer from 3.9.1 to 4.1.1 (b4e5d91)
- ci: Bump anchore/sbom-action from 0.23.1 to 0.24.0 (0d894c9)
- ci: Bump actions/cache from 5.0.3 to 5.0.4 (b884911)
- chore: update registry for v0.4.6 (7820eba)

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

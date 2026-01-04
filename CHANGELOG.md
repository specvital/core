# Changelog

## [1.5.0](https://github.com/specvital/core/compare/v1.4.0...v1.5.0) (2026-01-04)

### 🎯 Highlights

#### ✨ Features

- **cargotest:** detect test macros by analyzing same-file macro_rules! definitions ([4f3d697](https://github.com/specvital/core/commit/4f3d6975418aa9b381d8ebbdb59d429c119cbbcb))
- **junit4:** add JUnit 4 framework support ([7b96c63](https://github.com/specvital/core/commit/7b96c631594e7170273ff83358c4e32c16426854))
- **junit5:** add Java 21+ implicit class test detection ([d7c1218](https://github.com/specvital/core/commit/d7c1218c4fa8d8e6828b7940f4d6ed2507483d3a))
- **kotest:** detect tests defined in init blocks ([4a5a2d8](https://github.com/specvital/core/commit/4a5a2d81f294332cbf6b2dc30566923d8c338a11))
- **source:** add commit timestamp retrieval to GitSource ([afbe437](https://github.com/specvital/core/commit/afbe4372532ffeaff10c9a6f0622c2df70a881bc))
- **swift-testing:** add Apple Swift Testing framework support ([161b650](https://github.com/specvital/core/commit/161b650a186bdf2fc6a7e85d8b4303d7c3f84fc4))
- **vitest:** add test.for/it.for API support (Vitest 4.0+) ([5c7c8fa](https://github.com/specvital/core/commit/5c7c8fa6df22d18dda3257171617f68d1792a17d))

#### 🐛 Bug Fixes

- **cargo-test:** add macro-based test detection for Rust ([caa4d1b](https://github.com/specvital/core/commit/caa4d1bd76f2d42bfee1a48a3b01bbc21155ee83))
- **detection:** detect .cy.ts files as Cypress even within Playwright scope ([8ee2526](https://github.com/specvital/core/commit/8ee2526c4b1af920ff61410dbb5d9e31dbc0f96f))
- **dotnet:** detect tests inside C# preprocessor directives ([295f836](https://github.com/specvital/core/commit/295f836a750e4b1c8fb7813da03de58d9a1dc0e7))
- **dotnet:** support individual counting for parameterized test attributes ([e1d0d5f](https://github.com/specvital/core/commit/e1d0d5fff3b46490d4feec3b76498fdffeaa03b9))
- **gotesting:** support Test_xxx pattern test function detection ([fb7aeaf](https://github.com/specvital/core/commit/fb7aeafcbb5458cdef8ded2d2dba92a80d985c8a))
- **gtest:** add missing TYPED_TEST and TYPED_TEST_P macro detection ([cbb3914](https://github.com/specvital/core/commit/cbb391430c34fcd1117b7e263088eb553c068066))
- **gtest:** detect nested tests within tree-sitter ERROR nodes ([0ade3c7](https://github.com/specvital/core/commit/0ade3c7d443947918fa73861cc2910d0f998a5ea))
- **integration:** fix missing validation for multi-framework repositories ([5abc0a4](https://github.com/specvital/core/commit/5abc0a4c18bbc633d95a9c968c62a97e5eee7e3e))
- **jest:** support multiple root directories via Jest config roots array ([7e5bfea](https://github.com/specvital/core/commit/7e5bfeaee7940477d41f06265ee27a6400cd9347))
- **jstest:** add missing test detection in variable declarations ([d17f77d](https://github.com/specvital/core/commit/d17f77df86463b35c20ee13a5fbe4c6cbe22d5f8))
- **jstest:** count it.each/describe.each as single test ([fdbd484](https://github.com/specvital/core/commit/fdbd484070a159f24e988185cb3265358f75d5bb))
- **jstest:** detect jscodeshift defineTest calls as dynamic tests ([92bfb56](https://github.com/specvital/core/commit/92bfb5674afff7e89c3161fe1dc09f815b943c6e))
- **jstest:** detect tests inside IIFE conditional describe/it patterns ([1635945](https://github.com/specvital/core/commit/1635945c34026b2fadf4687f031cdb1f46790e6b))
- **jstest:** detect tests inside loop statements ([6c5b066](https://github.com/specvital/core/commit/6c5b066793971629fb7ed35f999b71e54c4833ec))
- **jstest:** detect tests using member expression as test name ([1570397](https://github.com/specvital/core/commit/157039798be41f72f258386c0026e642f4b53747))
- **jstest:** detect tests with variable names inside forEach callbacks ([efe2ec5](https://github.com/specvital/core/commit/efe2ec5b7dd9d2c3366519a24a6e698498770353))
- **jstest:** filter out Vitest conditional skip API from test detection ([3280998](https://github.com/specvital/core/commit/3280998a3fad6e900f2e2a3a0c85c95be07bb2ee))
- **jstest:** fix test detection failure in TSX files ([3d57940](https://github.com/specvital/core/commit/3d57940f70c76a3ed53904ae656088a2887950ff))
- **jstest:** fix test detection inside custom wrapper functions ([9c91958](https://github.com/specvital/core/commit/9c919581e4b8a3894a886f43737b7c0c32c7a572))
- **jstest:** support dynamic test detection in forEach/map callbacks and object arrays ([bc51894](https://github.com/specvital/core/commit/bc51894086b4d2730ea7f1dd78203e9e64f83ef9))
- **jstest:** support ESLint RuleTester.run() pattern detection ([b5e18f9](https://github.com/specvital/core/commit/b5e18f9f5dc975b25b12f11f1970ee3eb0f40903))
- **jstest:** support include/exclude pattern parsing in Jest/Vitest configs ([801e455](https://github.com/specvital/core/commit/801e45595aa0ec8d1a920bc396efc36a2aa9b716))
- **junit4:** detect tests inside nested static classes ([5673d83](https://github.com/specvital/core/commit/5673d83cee6c18c758adc5149b49bf411ea2ed83))
- **junit5:** add @TestFactory and @TestTemplate annotation support ([e868ef5](https://github.com/specvital/core/commit/e868ef56ddabd0ac85af83046c6e6b6c969c6eed))
- **junit5:** add custom @TestTemplate-based annotation detection ([242e320](https://github.com/specvital/core/commit/242e320505a7dfa7db664187babcc2e22f6d2b6f))
- **junit5:** detect Kotlin test files ([9090b51](https://github.com/specvital/core/commit/9090b51147cdd6ee4947ae8c02ed46fb666293a6))
- **junit5:** exclude JUnit4 test files from JUnit5 detection ([02aaed1](https://github.com/specvital/core/commit/02aaed191af6b47097f501f35d0db9421d53d79d))
- **junit5:** exclude src/main path from test file detection ([7e5ce26](https://github.com/specvital/core/commit/7e5ce26df4fdd5976c3874dd37130e6f4363497e))
- **kotest:** add missing WordSpec, FreeSpec, ShouldSpec style parsing ([424ab3a](https://github.com/specvital/core/commit/424ab3aa159fe13b9c132fa9117324cd0e5050d1))
- **kotest:** detect tests inside forEach/map chained calls ([cbd1fb1](https://github.com/specvital/core/commit/cbd1fb1bd85d39a1c0447a59dea58c05fb2624f5))
- **minitest:** resolve Minitest files being misdetected as RSpec ([93305d9](https://github.com/specvital/core/commit/93305d997c3cfdbd9dc945b59e022f9d611a682b))
- **mstest:** support C# file discovery under test/ directory ([95bcc31](https://github.com/specvital/core/commit/95bcc31e4f11aeee70b7bf00f52a91775f696618))
- **parser:** handle NULL bytes in source files that caused test detection failure ([d9f959c](https://github.com/specvital/core/commit/d9f959cf101cef4bdd11603650a5018a34894217))
- **phpunit:** add missing indirect inheritance detection for \*Test suffix ([106b73d](https://github.com/specvital/core/commit/106b73d5aa1c8cc4354d915d6bbe481044f655c8))
- **playwright:** detect conditional skip API calls as non-test ([129f0c0](https://github.com/specvital/core/commit/129f0c09ad8bc63c6b942c67ba8154874884aa3f))
- **playwright:** detect indirect import tests even with import type present ([e353cac](https://github.com/specvital/core/commit/e353cac3091a344f452e797c507a9cfd7483adfc))
- **playwright:** detect indirect imports and test.extend() patterns ([aa22e18](https://github.com/specvital/core/commit/aa22e18b15173d801805856dd4364bbdca55bbcb))
- **playwright:** fix config scope-based framework detection bug ([4983492](https://github.com/specvital/core/commit/4983492e1e4ac0908892f9764c8efde5f49e8d61))
- **playwright:** support test function detection with import aliases ([4ba46b7](https://github.com/specvital/core/commit/4ba46b752406b8439e15ea2aba2f04d07841c60f))
- **pytest:** fix unittest.mock imports being misclassified as unittest ([4ad41ed](https://github.com/specvital/core/commit/4ad41ed450747445ed3928619d6a2a9bf90e2352))
- **rspec:** detect tests inside loop blocks (times, each, etc.) ([a68b270](https://github.com/specvital/core/commit/a68b2705d793f145bfd4fff88419f464e5ad615a))
- **rspec:** resolve RSpec files being misdetected as Minitest ([b21dda9](https://github.com/specvital/core/commit/b21dda9a43b7d3bf5d70c221543c832bc3373aab))
- **scanner:** exclude **fixtures**, **mocks** directories from scan ([881f360](https://github.com/specvital/core/commit/881f36037d25cb6583b2e1fa30d3e715435801a6))
- **scanner:** fix symlink duplicate counting and coverage pattern bug ([aba78d1](https://github.com/specvital/core/commit/aba78d169a1f7909307687c67c45d67c19a84b99))
- **scanner:** use relative path instead of absolute path for test file detection ([e1937d2](https://github.com/specvital/core/commit/e1937d281ab27136b0df8847b8fec84a8de4baa3))
- **testng:** add missing class-level @Test annotation detection ([7912275](https://github.com/specvital/core/commit/79122754e9a3d413ebb69836434f9bad7bc5ab78))
- **testng:** detect @Test inside nested classes ([aaad38e](https://github.com/specvital/core/commit/aaad38ed4ff83d1f48e130935acea6f9bc934282))
- **xunit:** support custom xUnit test attributes (*Fact, *Theory) ([4845628](https://github.com/specvital/core/commit/48456284a5d5df720a05a1b7debfaaa22e1d977c))

### 🔧 Maintenance

#### 📚 Documentation

- **dotnetast:** document tree-sitter-c-sharp preprocessor limitation ([104ec57](https://github.com/specvital/core/commit/104ec5728101e4fed8fcb3593b61dbfc245bcbae))
- **validate-parser:** add ADR policy review step ([06e46ec](https://github.com/specvital/core/commit/06e46ec548e4c320f303cc004cade92e14261472))
- **validate-parser:** allow repos.yaml repos on explicit request and enforce Korean report ([a9c1852](https://github.com/specvital/core/commit/a9c18529a9e1f5a396cef46a0bae3622357219b1))

#### ✅ Tests

- add integration test case - kubrickcode/baedal ([b81e4ac](https://github.com/specvital/core/commit/b81e4ac432735abc4afeb1dcba1e7ee84b77b038))
- add integration test case - specvital/collector ([ba5e703](https://github.com/specvital/core/commit/ba5e703d7087fe3502e5368317ff054658087176))
- add integration test case - specvital/core ([8523567](https://github.com/specvital/core/commit/85235670ae9407086433b056c83226e23a05c7bb))
- add integration test case - specvital/web ([17b455f](https://github.com/specvital/core/commit/17b455fc05076c1e1e18ce080ca57156680c5a90))
- add test case - kubrickcode/quick-command-buttons ([14c93c6](https://github.com/specvital/core/commit/14c93c680b85b581e9d987d43313da2a4b908c01))
- **junit5:** update integration snapshots for Kotlin support ([3e9aa54](https://github.com/specvital/core/commit/3e9aa548ae730819731dbecebba1bf2b982fbfe9))

#### 🔨 Chore

- add custom commands for parser validation ([c8bd024](https://github.com/specvital/core/commit/c8bd0246c1b466000b1049805254e9954212b6c4))
- add flask as integration test ([cf63e7c](https://github.com/specvital/core/commit/cf63e7c879c01440f458fcf0876da7d7b62d7690))
- fix vscode import area not automatically collapsing ([218fb9e](https://github.com/specvital/core/commit/218fb9e5f9e93ceef583237b10854ac3fb6d546e))
- **integration:** update cypress test repository to v15.8.1 ([36d6040](https://github.com/specvital/core/commit/36d60408103d0310c74a62d85dc7c473e2275a18))
- setting up devcontainers for testing various languages ​​and frameworks ([0f3b08e](https://github.com/specvital/core/commit/0f3b08ec5ddb18655b70ccdf56f081cdddc71a5f))
- snapshot update ([601530e](https://github.com/specvital/core/commit/601530e22796d92101eeb7deada55c351108dc2b))
- sync docs ([d8ec48c](https://github.com/specvital/core/commit/d8ec48c4b2757bda51c8637d1afb420390541577))
- sync docs ([4716fb2](https://github.com/specvital/core/commit/4716fb2f67a5ce77d87caa71734054dabee57070))
- sync docs ([bd09a40](https://github.com/specvital/core/commit/bd09a40a6b72689e7b3579f10467c85f35d163b1))
- sync docs ([ba18e47](https://github.com/specvital/core/commit/ba18e4780f7ca7c1da8370d6f43f98b6e4c8bd97))
- sync docs ([ae6a331](https://github.com/specvital/core/commit/ae6a331bbf8cf291acab6c6b6ba44960766c2367))
- sync docs ([4f00b6c](https://github.com/specvital/core/commit/4f00b6c4004d7fb23cbbd5e75b0f507a1294c4a1))

## [1.4.0](https://github.com/specvital/core/compare/v1.3.0...v1.4.0) (2025-12-20)

### 🎯 Highlights

#### ✨ Features

- **crypto:** add NaCl SecretBox encryption package ([2bab1b3](https://github.com/specvital/core/commit/2bab1b313d720e7dcea1a148db6516202b25c035))
- **gotesting:** add Benchmark/Example/Fuzz function support ([76296d5](https://github.com/specvital/core/commit/76296d5d019cd91ee0cdd23f838d5a6a20c86494))
- **jstest:** add Jest/Vitest concurrent modifier support ([704b25c](https://github.com/specvital/core/commit/704b25c5e11972d86a6502aab918d873b8b7ec03))
- **mocha:** add Mocha TDD interface support ([2348b66](https://github.com/specvital/core/commit/2348b66975275675f88be5e96938ee73007774fc))
- **vitest:** add bench() function support ([b1f8949](https://github.com/specvital/core/commit/b1f89495c596f5ea6ae176200622a4678506ac0a))

#### 🐛 Bug Fixes

- disable implicit credential helper in git operations ([08f42b2](https://github.com/specvital/core/commit/08f42b24ef3fa0a889025f9ee08364f1d1eaa380))

### 🔧 Maintenance

#### 📚 Documentation

- add missing version headers and improve CHANGELOG hierarchy ([f38e681](https://github.com/specvital/core/commit/f38e6815e707bc2e0f91813cc5e29496c1349a3d))
- edit CLAUDE.md ([9ab5494](https://github.com/specvital/core/commit/9ab54945ab2d368e1f3a28b178152757728200ed))
- update README.md ([e76202e](https://github.com/specvital/core/commit/e76202ea4ecfae71cc2d077a85b88841b58e5f34))

#### 💄 Styles

- sort justfile ([78bacfd](https://github.com/specvital/core/commit/78bacfdc08f5b94ec6d1e5ef2aa253b7e52f2d6c))

#### 🔨 Chore

- ai-config-toolkit sync ([7d219c3](https://github.com/specvital/core/commit/7d219c3f04439b04609740456ac3e329c023e41c))
- changing the environment variable name for accessing GitHub MCP ([2079973](https://github.com/specvital/core/commit/20799738fab4dff39e53cd8b333b41d576afc465))
- delete unused mcp ([c3d1551](https://github.com/specvital/core/commit/c3d15516e01d0ef63770a0ef76eb55d641ab02af))
- **deps-dev:** bump @semantic-release/commit-analyzer ([9595ec8](https://github.com/specvital/core/commit/9595ec87709f40f1773cd1f90615295bde5b6baf))
- **deps-dev:** bump @semantic-release/github from 11.0.1 to 12.0.2 ([e05ee45](https://github.com/specvital/core/commit/e05ee45879c7d73ab619fe91bb7f064629aacfca))
- **deps-dev:** bump conventional-changelog-conventionalcommits ([9a13ed8](https://github.com/specvital/core/commit/9a13ed8541d955d77774762daab4d9834c1e863c))
- **deps:** bump actions/cache from 4 to 5 ([51d8d2b](https://github.com/specvital/core/commit/51d8d2b6c2ab69154f3ed645851bb7bb758d7465))
- **deps:** bump actions/checkout from 4 to 6 ([9bd5b6c](https://github.com/specvital/core/commit/9bd5b6c1feeadd1f81031f90457b0e8395053fea))
- **deps:** bump actions/setup-go from 5 to 6 ([12a3121](https://github.com/specvital/core/commit/12a31213d48b800004d9ec8d461c552a912eb94b))
- **deps:** bump actions/setup-node from 4 to 6 ([361f566](https://github.com/specvital/core/commit/361f5662b355e37b7e1be264ccf1c3059a13e0a5))
- **deps:** bump extractions/setup-just from 2 to 3 ([0fedea4](https://github.com/specvital/core/commit/0fedea40fa85390bc1c5a38873084dfa431a4790))
- **deps:** bump github.com/bmatcuk/doublestar/v4 from 4.8.1 to 4.9.1 ([d9f058f](https://github.com/specvital/core/commit/d9f058f8fef22cdb71de3db0e7d8191ef631834a))
- **deps:** bump github.com/stretchr/testify from 1.9.0 to 1.11.1 ([ab0bc83](https://github.com/specvital/core/commit/ab0bc8337c549d1e6fb44cbb98339947b0d99511))
- **deps:** bump golang.org/x/sync from 0.18.0 to 0.19.0 ([2b0070b](https://github.com/specvital/core/commit/2b0070bad52dfdf97102ef125c68eb0cdbf43417))
- Global document synchronization ([06c079c](https://github.com/specvital/core/commit/06c079c1487222eff2d771fc6db4d259a5bf273d))
- improved the claude code status line to display the correct context window size. ([365a13e](https://github.com/specvital/core/commit/365a13e86cd78593f63c51caea6f9df97dbbb200))
- modified container structure to support codespaces ([9e02cd4](https://github.com/specvital/core/commit/9e02cd44033a17317ef40f7e438eac1f0f013dcd))
- snapshot update ([78579ac](https://github.com/specvital/core/commit/78579accc74ec0596c3e1fae971fe33cb1da3e1e))
- snapshot update ([053ce8a](https://github.com/specvital/core/commit/053ce8a74203585873f6b578706cc7593e16511f))
- snapshot-update ([7c2fb1c](https://github.com/specvital/core/commit/7c2fb1c052e0af70880bde453622f25af0fb2410))
- sync ai-config-toolkit ([b7b852a](https://github.com/specvital/core/commit/b7b852ae34a6c46f0eb24471cb47fad528f13c77))
- sync ai-config-toolkit ([0b95b2d](https://github.com/specvital/core/commit/0b95b2d2bbc600fb23db92e7ddaabf41fbcb2957))

## [1.3.0](https://github.com/specvital/core/compare/v1.2.2...v1.3.0) (2025-12-11)

### 🎯 Highlights

#### ✨ Features

- **cypress:** add Cypress E2E testing framework support ([b87f92b](https://github.com/specvital/core/commit/b87f92b68e0e0263f9571a238693e6a96390c232))
- **gtest:** add C++ Google Test framework support ([3821565](https://github.com/specvital/core/commit/382156588fe30534679fb24e7005cd45be1315e7))
- **kotest:** add Kotlin Kotest test framework support ([374b696](https://github.com/specvital/core/commit/374b6963d5d9a666297c450ec2d8aeecb8adfca9))
- **minitest:** add Ruby Minitest test framework support ([2cf2c22](https://github.com/specvital/core/commit/2cf2c22eb6fb3964b3611ee72f12d50a96cb4566))
- **mocha:** add Mocha JavaScript test framework support ([fdbb49e](https://github.com/specvital/core/commit/fdbb49e7d7d048238da0e1af53e1f4ed3d24b627))
- **mstest:** add C# MSTest test framework support ([9ab6565](https://github.com/specvital/core/commit/9ab6565c8141a0b8fe4a80eb197da76432306bb0))
- **phpunit:** add PHP PHPUnit framework support ([d395cd5](https://github.com/specvital/core/commit/d395cd5627a94e54eeaee989c0e48817810f79ca))
- **testng:** add Java TestNG test framework support ([3c50d31](https://github.com/specvital/core/commit/3c50d318b58031b0c833c6872de8c248aa97cacb))
- **xctest:** add Swift XCTest test framework support ([7c62c95](https://github.com/specvital/core/commit/7c62c95a2f4b7120b938daf7f6e09466479f3d23))

### 🔧 Maintenance

#### ♻️ Refactoring

- delete deprecated code ([c55a9ac](https://github.com/specvital/core/commit/c55a9ac53695ae0e4bfac1f5c70d9c20b9a20239))
- remove dead code from MVP development ([3a08c21](https://github.com/specvital/core/commit/3a08c21fe0e15ed11a793bb2a7943b1c53736191))

#### 🔨 Chore

- add missing framework constants ([324bad0](https://github.com/specvital/core/commit/324bad03d66c25fcaeb4ee6be0859e2dfa5e2607))

## [1.2.2](https://github.com/specvital/core/compare/v1.2.1...v1.2.2) (2025-12-10)

### 🔧 Maintenance

#### 🔧 Internal Fixes

- **release:** fix 404 error on commit links in release notes ([3bcff5e](https://github.com/specvital/core/commit/3bcff5e9498bec9aa56edbb9797d51263888088b))

## [1.2.1](https://github.com/specvital/core/compare/v1.2.0...v1.2.1) (2025-12-10)

### 🔧 Maintenance

#### 🔧 Internal Fixes

- **release:** fix broken commit links and long hash display in release notes ([fe38507](https://github.com/specvital/core/commit/fe3850790f60df701af655b4e7177899bfcb80ff))

#### 🔨 Chore

- adding recommended extensions ([328447f](https://github.com/specvital/core/commit/328447f811601b35b6ca2e71c3bf83183a77af35))

## [1.2.0](https://github.com/specvital/core/compare/v1.1.2...v1.2.0) (2025-12-10)

### 🎯 Highlights

#### ✨ Features

- add all package for bulk parser strategy registration ([96ffbe6](https://github.com/specvital/core/commit/96ffbe688e750a18df1556b9e41157f4a0d4306e))
- add C# language and xUnit test framework support ([3b3c685](https://github.com/specvital/core/commit/3b3c685c6fb0f26cbf4b6865a0dd64f2231fac55))
- add GitSource implementation for remote repository access ([c8743a5](https://github.com/specvital/core/commit/c8743a5872ea641f2916c035c57f88960122da77))
- add Java language and JUnit 5 test framework support ([cc1a6ba](https://github.com/specvital/core/commit/cc1a6ba153bc5e9ea4c5af9e5c2672a2cd9020a7))
- add NUnit test framework support for C# ([b62c420](https://github.com/specvital/core/commit/b62c4208777cbc43f8e63af370ef0c9c01636f39))
- add Python pytest framework support ([b153129](https://github.com/specvital/core/commit/b153129a6dc81e3c56f90c33a03731d45cee5b1c))
- add Python unittest framework support ([bcac628](https://github.com/specvital/core/commit/bcac628e882152a610c2d898c93ac9e5824c642e))
- add Ruby language and RSpec test framework support ([3e28c47](https://github.com/specvital/core/commit/3e28c476c8f338b8ab25e08774feed4ca272fd5e))
- add Source interface and LocalSource implementation ([af0e2ed](https://github.com/specvital/core/commit/af0e2ed2e49c51bfc5d788f7e96397a862a8850a))
- **domain:** add Modifier field to Test/TestSuite ([a1b9363](https://github.com/specvital/core/commit/a1b93633275941ddd3b4fee7db46327a656eab21))
- **parser:** add Rust cargo test framework support ([30feca7](https://github.com/specvital/core/commit/30feca749a0d9c64b4eccb7ea6ed5a66c3ab4516))
- **source:** add Branch method to GitSource ([8d6f10d](https://github.com/specvital/core/commit/8d6f10d556506732d3b00750352889f992e9520e))
- **source:** add CommitSHA method to GitSource ([97256ec](https://github.com/specvital/core/commit/97256ec27766fdba1fb67942fc1d58fe4252f36f))
- **vitest:** add VitestContentMatcher for vi.\* pattern detection ([9d2c72e](https://github.com/specvital/core/commit/9d2c72e8fdf71bfca654e7835be902c05e698862))

#### 🐛 Bug Fixes

- **detection:** fix Go test files not being detected ([8487f71](https://github.com/specvital/core/commit/8487f71642be502c3a6ba66ba29398bae273d42b))
- **detection:** fix scope-based framework detection bugs ([3589928](https://github.com/specvital/core/commit/35899280fbf45a7ec8dae2987a51fb48143adef2))
- **parser:** prevent slice bounds panic in tree-sitter node text extraction ([465e9bc](https://github.com/specvital/core/commit/465e9bc0d0aeee688c29ab786ddeafbb88d76d87))
- **tspool:** fix flaky tests caused by tree-sitter parser reuse ([256c9aa](https://github.com/specvital/core/commit/256c9aa1780471334ee0d28ede877b050a5cc2d6))

### 🔧 Maintenance

#### 🔧 Internal Fixes

- fix nondeterministic integration test results ([41e3d38](https://github.com/specvital/core/commit/41e3d3831892ca52c59e621d75172651ca0ecbdc))

#### 💄 Styles

- format code ([71d8f66](https://github.com/specvital/core/commit/71d8f66631e6fb29e55e9d3ea934806e1a1b806f))

#### ♻️ Refactoring

- change Scanner to read files through Source interface ([11507ac](https://github.com/specvital/core/commit/11507accf9d0a9f34e18cb8bdaf80e62f6333c5e))
- **detection:** redesign with unified framework definition system ([9ba32af](https://github.com/specvital/core/commit/9ba32af300f73bf08746ac24e3fcb4ea48d5291b))
- **detection:** replace score accumulation with early-return approach ([ab30e72](https://github.com/specvital/core/commit/ab30e72e4d2a2bcb4d45baed9eac8cc422286ba5))
- **domain:** align TestStatus constants with DB schema ([babec36](https://github.com/specvital/core/commit/babec3602a02ece88b8a22b8729f335a96163555))

#### ✅ Tests

- add 8 complex case repositories for edge case coverage ([619f361](https://github.com/specvital/core/commit/619f361801a76059e7e4f7e8206a1486e67de420))
- add golden snapshot comparison to integration tests ([1cffd01](https://github.com/specvital/core/commit/1cffd019a34302a9f4d253cda11d6868d6fe61f9))
- add integration test infrastructure with real GitHub repos ([476b3eb](https://github.com/specvital/core/commit/476b3eb16953add6a64023f64bb68aa4de8e841f))
- add unittest integration test repositories ([7d31dcf](https://github.com/specvital/core/commit/7d31dcfa256d9106bf831e595831e35722b5e72e))

#### 🔧 CI/CD

- add integration test CI workflow and documentation ([d9368e1](https://github.com/specvital/core/commit/d9368e181da2745c02b007652c00694dc88b0d7d))

#### 🔨 Chore

- add snapshot-update command and refresh golden snapshots ([c3e47e8](https://github.com/specvital/core/commit/c3e47e8bf274eef00f0088a4a13d7a66a24c072b))
- add useful action buttons ([ef1a60c](https://github.com/specvital/core/commit/ef1a60cd9f88ca7457e366bc1978c03750019316))
- ai-config-toolkit sync ([e631a30](https://github.com/specvital/core/commit/e631a30fde776b9ba023ec00989cf2a8605e39d6))
- ai-config-toolkit sync ([42eeba3](https://github.com/specvital/core/commit/42eeba3426c41231ebefa9fc431fd3884f954b2d))
- snapshot update ([f4c171d](https://github.com/specvital/core/commit/f4c171dbf86a02cfa471806d1da98f014899c161))
- sync integration repos ([02c6a8d](https://github.com/specvital/core/commit/02c6a8d4311bcae40bca218e8a2081b8392a4755))
- sync snapshot ([6c086e9](https://github.com/specvital/core/commit/6c086e9c4297b074b7184678c3fde40a5bbdc00f))

## [1.1.2](https://github.com/specvital/core/compare/v1.1.1...v1.1.2) (2025-12-05)

### 🎯 Highlights

#### 🐛 Bug Fixes

- **detection:** fix glob patterns being incorrectly treated as comments ([85fd875](https://github.com/specvital/core/commit/85fd875d706cd1330fd0b8a27f3d1514f36e4013))

## [1.1.1](https://github.com/specvital/core/compare/v1.1.0...v1.1.1) (2025-12-05)

### 🎯 Highlights

#### 🐛 Bug Fixes

- **detection:** add ProjectContext for source-agnostic framework detection ([708f70a](https://github.com/specvital/core/commit/708f70aac041918ea7ff41d698fca45e43d6809d))

## [1.1.0](https://github.com/specvital/core/compare/v1.0.3...v1.1.0) (2025-12-05)

### 🎯 Highlights

#### ✨ Features

- **parser:** add hierarchical test framework detection system ([7655868](https://github.com/specvital/core/commit/76558682788612995f762de422d965f4fa2836ad))

### 🔧 Maintenance

#### 🔨 Chore

- add useful action buttons ([eb2b93b](https://github.com/specvital/core/commit/eb2b93b8e163c2e538a025cff0e35abad891a87b))
- delete unused file ([d6f2203](https://github.com/specvital/core/commit/d6f220316bd8e66423366f61a153627aa0daa7bd))
- syncing documents from ai-config-toolkit ([1faaf43](https://github.com/specvital/core/commit/1faaf4364d1782493008d8abbae66283d35861af))

## [1.0.3](https://github.com/specvital/core/compare/v1.0.2...v1.0.3) (2025-12-04)

### 🎯 Highlights

#### 🐛 Bug Fixes

- **gotesting:** fix Go test parser incorrectly returning pending status ([14f1336](https://github.com/specvital/core/commit/14f133635410d9ced0d747d7245238e84f6014c9))

### 🔧 Maintenance

#### 📚 Documentation

- sync CLAUDE.md ([167df5b](https://github.com/specvital/core/commit/167df5b587fbbaa8b6ade0dbb4c0ecc0ea41fb98))

#### 🔨 Chore

- add auto-formatting to semantic-release pipeline ([f185576](https://github.com/specvital/core/commit/f185576d2247234c46ec1c0027c8898a775ef5cd))

## [1.0.2](https://github.com/specvital/core/compare/v1.0.1...v1.0.2) (2025-12-04)

### 🔧 Maintenance

#### 🔧 Internal Fixes

- fix Go module zip creation failure ([3ceb7d6](https://github.com/specvital/core/commit/3ceb7d626ead57835083b0c45d2c7091cb62757f))

## [1.0.1](https://github.com/specvital/core/compare/v1.0.0...v1.0.1) (2025-12-04)

### 🔧 Maintenance

#### 🔧 Internal Fixes

- exclude unnecessary files from Go module zip ([0e3f8fa](https://github.com/specvital/core/commit/0e3f8fa9598ce226632139c2b18dd4d710ad79af))

## [1.0.0](https://github.com/specvital/core/releases/tag/v1.0.0) (2025-12-04)

### 🎯 Highlights

#### ✨ Features

- add Go test parser support ([3e147a5](https://github.com/specvital/core/commit/3e147a59b2ec6799db588702a648fd25bb3d44c0))
- add parallel test file scanner with worker pool ([d8dbe13](https://github.com/specvital/core/commit/d8dbe13cc5095a4c2385add15c320c1f9148f76d))
- add Playwright test parser support ([c779d70](https://github.com/specvital/core/commit/c779d7063fdc58e60b085e9daf21b2a8453db7b0))
- add test file detector for automatic discovery ([a71bec4](https://github.com/specvital/core/commit/a71bec4e61c6a05406b9021e6ebd929dce4fff05))
- add Vitest test parser support ([d4226f5](https://github.com/specvital/core/commit/d4226f5238edb8131074fe22cd54d492eae70a94))
- implement Jest test parser core module ([caffaab](https://github.com/specvital/core/commit/caffaab77d810283a126266ed806f4bb1bdc2a0a))

#### ⚡ Performance

- add parser pooling and query caching for concurrent parsing ([e8ff8f4](https://github.com/specvital/core/commit/e8ff8f40ddecd3143d56ecc97c78075e112806cd))

### 🔧 Maintenance

#### 📚 Documentation

- add GoDoc comments and library usage guide ([72f5220](https://github.com/specvital/core/commit/72f5220e7ab96b1497fedfe4f59230774cefe369))

#### ♻️ Refactoring

- move go.mod to root to enable external imports ([8976869](https://github.com/specvital/core/commit/89768699151849542582997faa26ec9d6557e923))
- move src/pkg to pkg for standard Go layout ([3ed1d78](https://github.com/specvital/core/commit/3ed1d782be5b6e3d2fcfbeb45aa003ad48e2eb10))

#### 🔧 CI/CD

- configure semantic-release based automated release pipeline ([3e85cee](https://github.com/specvital/core/commit/3e85ceeb26ca91009c7c76dc71108ef985ea9538))

#### 🔨 Chore

- **deps-dev:** bump lint-staged from 15.2.11 to 16.2.7 ([94b8012](https://github.com/specvital/core/commit/94b801204734591d3a8aaece07562ae0423354b7))

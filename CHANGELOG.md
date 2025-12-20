# Changelog

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

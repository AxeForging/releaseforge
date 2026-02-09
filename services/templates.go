package services

var BuiltinTemplates = map[string]string{
	"semver-release-notes": semverReleaseNotes,
	"conventional-changelog": conventionalChangelog,
	"version-analysis":       versionAnalysis,
}

const semverReleaseNotes = ` ## Summary
- A brief summary of the release

 ## Breaking Changes
- List of breaking changes that require user action

 ## Features
- List of new features and enhancements

 ## Improvements
- List of improvements and optimizations

 ## Fixes
- List of bug fixes and patches

 ## Security
- List of security updates and fixes if any

 ## Documentation
- List of documentation updates

 ### Updated (dependencies)
- List of updated/added/removed dependencies

 ## Technical Details
- List of summarized commit messages (a title for each commit message) with their refs (ref here)

 ## Contributors
- List of contributors to this release

 ## Testing
- Testing notes and considerations`

const conventionalChangelog = `# Conventional Changelog Template

## Summary
A brief summary of the release

## Changes

### Features
List of new features (conventional commit type: feat)

### Fixes
List of bug fixes (conventional commit type: fix)

### Documentation
List of documentation changes (conventional commit type: docs)

### Style
List of style changes (conventional commit type: style)

### Refactoring
List of refactoring changes (conventional commit type: refactor)

### Performance
List of performance improvements (conventional commit type: perf)

### Testing
List of test changes (conventional commit type: test)

### Build
List of build system changes (conventional commit type: build)

### CI/CD
List of CI/CD changes (conventional commit type: ci)

### Chores
List of maintenance tasks (conventional commit type: chore)

## Breaking Changes
List of breaking changes (commits with !: or BREAKING CHANGE:)

## Security
List of security-related changes

## Dependencies

### Added
List of new dependencies

### Updated
List of updated dependencies with version changes

### Removed
List of removed dependencies

## Migration Guide
Step-by-step migration instructions from previous version

## Contributors
List of contributors with their contributions

## Deployment Notes
Special deployment considerations

## Suggested Version
The suggested next version based on commit analysis`

const versionAnalysis = `# Version Analysis Template

## Summary
Analysis of commits to determine appropriate version bump

## Commit Analysis

### Breaking Changes
List of commits that introduce breaking changes

### New Features
List of commits that add new features

### Bug Fixes
List of commits that fix bugs

### Improvements
List of commits that improve existing functionality

### Documentation
List of commits that update documentation

### Maintenance
List of maintenance and chore commits

## Version Recommendation

### Suggested Version
The recommended next version

### Reasoning
Explanation for the version recommendation

### Impact Assessment
Assessment of the impact of this release

## Migration Notes
Any migration steps required for this version

## Suggested Version
The suggested next version based on commit analysis`

# Release Guide

This guide covers how to build, tag, publish, and verify a Mattermost Flow plugin release.

## Versioning Model

The plugin version is injected at build time from Git tags.

- If the current commit has a tag such as `v0.1.2`, the bundled plugin version becomes `0.1.2`
- If the commit is not tagged, the bundle version is derived from the nearest tag plus the current commit hash

Because of this, release builds should come from a clean, intentional Git tag.

## Local Build

Create a release bundle locally:

```bash
make dist
```

Output:

```text
dist/com.mattermost.flow-plugin-<version>.tar.gz
```

## Tag-Based GitHub Release Flow

This repository uses GitHub Actions to publish releases automatically.

1. Push the latest commit to `main`
2. Create an annotated release tag
3. Push the tag to GitHub
4. GitHub Actions builds the plugin bundle, generates `SHA256SUMS.txt`, and publishes both assets to a GitHub Release

Example:

```bash
git tag -a v0.1.2 -m "Mattermost Flow Plugin v0.1.2"
git push origin v0.1.2
```

## Helper Targets

The `Makefile` includes convenience targets for semantic version tags:

```bash
make patch
make minor
make major
make patch-rc
make minor-rc
make major-rc
```

These helpers are intended to be run from `main` or a release branch and will push the generated tag.

## Executable Permissions

The release bundler writes files under `server/dist/` into the archive with mode `0755`.

This matters because Mattermost plugin bundles must contain runnable server binaries after extraction. The custom bundle step avoids the common problem where the plugin uploads successfully but fails to start because executable bits were lost in the archive.

Relevant implementation files:

- [build/manifest/main.go](../build/manifest/main.go)
- [build/package_plugin.ps1](../build/package_plugin.ps1)
- [Makefile](../Makefile)

## Verifying a Release

After a tag push:

1. Confirm the release workflow succeeds in GitHub Actions
2. Confirm the GitHub Release exists
3. Confirm both assets are present:
   - `com.mattermost.flow-plugin-<version>.tar.gz`
   - `SHA256SUMS.txt`
4. Optionally verify the SHA256 checksum locally

## Rollback

If a release is bad:

- Disable the plugin in Mattermost
- Reinstall a previous release bundle
- Re-enable the plugin

Because the plugin stores state in KV, validate rollback behavior in staging when the release contains data model changes.

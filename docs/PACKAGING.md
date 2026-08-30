# Updates and packages

Landing in the official Debian/Ubuntu archive (`apt install` from `archive.ubuntu.com`) is a long process: ITP, a sponsor maintainer, Debian Policy, review. You do not need that for your own VPS fleet.

The practical ladder:

1. **Now:** binary from GitHub Releases and `gamayun --update`.
2. **Already in place:** a `.deb` in the same release → `apt install ./gamayun_x.y.z_amd64.deb`.
3. **Later:** your own apt repository (then updates are `apt upgrade`).

## Updating an installed agent

If you installed via `install.sh` or copied the binary yourself:

```bash
sudo gamayun --update
```

The command looks at the latest release, verifies SHA256, replaces the binary, and runs `systemctl restart`. Optional `GITHUB_TOKEN` is sent as a Bearer token to GitHub (API and assets) to avoid unauthenticated rate limits. Where the repository comes from, in this order:

1. `--repo owner/name` flag
2. what was baked into the binary at CI build time (`GITHUB_REPOSITORY`)
3. `update.github_repo` in `/etc/gamayun/config.yaml`
4. `GAMAYUN_REPO` environment variable

If the package is already installed via apt, `--update` refuses and asks for `apt upgrade` — the two update channels must not fight.

```bash
gamayun --version
```

## .deb without your own repository

After a `v*` tag:

```bash
wget https://github.com/voronkovd/gamayun/releases/download/v1.0.0/gamayun_1.0.0_amd64.deb
sudo apt install ./gamayun_1.0.0_amd64.deb
```

`postinst` does not overwrite an existing `/etc/gamayun/config.yaml`.

Build the package locally (needs `dpkg-deb`, i.e. Linux):

```bash
make deb VERSION=v1.0.0 REPO=owner/name
```

## Your own apt repository (when you need it)

Not "get into Ubuntu" — ship the package to your own machines via `apt update && apt upgrade`.

Minimum set:

- the release already has `.deb` files for amd64 and arm64;
- a separate job or script puts them into a repo (`reprepro`, `aptly`, or [charmbracelet/soft-serve](https://github.com/charmbracelet/soft-serve) is optional — `reprepro` is enough);
- hosting: GitHub Pages, Cloudflare R2, or plain nginx on one VPS;
- a GPG key that signs `Release`;
- on each machine, one file:

```
# /etc/apt/sources.list.d/gamayun.list
deb [signed-by=/usr/share/keyrings/gamayun.gpg] https://example.com/apt stable main
```

After that, updates are a normal `apt upgrade`. You can leave `--update` alone: it will yield to apt.

Hosted options if you do not want to run reprepro: [Cloudsmith](https://cloudsmith.io), [Gemfury](https://gemfury.com), [Packagecloud](https://packagecloud.io).

## Official Debian/Ubuntu

Worth it only if thousands of other people's machines need the package. Then: an ITP on bugs.debian.org, a Policy-compliant package, a Debian sponsor. That takes months and is not about monitoring your own fleet.

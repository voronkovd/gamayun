# Обновление и пакеты

Официальный архив Debian/Ubuntu (`apt install` из `archive.ubuntu.com`) — это отдельный долгий процесс: ITP, спонсор-мейнтейнер, Debian Policy, ревью. Для своего флота VPS это не нужно.

Рабочая лестница такая:

1. **Сейчас:** бинарник с GitHub Releases и `gamayun --update`.
2. **Уже заложено:** `.deb` в том же релизе → `apt install ./gamayun_x.y.z_amd64.deb`.
3. **Потом:** свой apt-репозиторий (тогда обновление = `apt upgrade`).

## Обновление установленного агента

Если ставили через `install.sh` или просто скопировали бинарник:

```bash
sudo gamayun --update
```

Команда смотрит latest release, сверяет SHA256, подменяет бинарник и делает `systemctl restart`. Откуда брать репозиторий, в таком порядке:

1. флаг `--repo owner/name`
2. то, что зашили в бинарник при сборке CI (`GITHUB_REPOSITORY`)
3. `update.github_repo` в `/etc/gamayun/config.yaml`
4. переменная `GAMAYUN_REPO`

Если пакет уже стоит через apt, `--update` откажется и попросит `apt upgrade` — два канала обновления не должны драться.

```bash
gamayun --version
```

## .deb без своего репозитория

После тега `v*`:

```bash
wget https://github.com/voronkovd/gamayun/releases/download/v1.0.0/gamayun_1.0.0_amd64.deb
sudo apt install ./gamayun_1.0.0_amd64.deb
```

`postinst` не затирает существующий `/etc/gamayun/config.yaml`.

Собрать пакет локально (нужен `dpkg-deb`, то есть Linux):

```bash
make deb VERSION=v1.0.0 REPO=owner/name
```

## Свой apt-репозиторий (когда понадобится)

Не «попасть в Ubuntu», а раздать пакет своим машинам через `apt update && apt upgrade`.

Минимальный набор:

- в релизе уже есть `.deb` для amd64 и arm64;
- отдельный job или скрипт кладёт их в repo (`reprepro`, `aptly` или [charmbracelet/soft-serve](https://github.com/charmbracelet/soft-serve) не обязателен — достаточно `reprepro`);
- хостинг: GitHub Pages, Cloudflare R2, обычный nginx на одном VPS;
- GPG-ключ, которым подписывается `Release`;
- на машинах один файл:

```
# /etc/apt/sources.list.d/gamayun.list
deb [signed-by=/usr/share/keyrings/gamayun.gpg] https://example.com/apt stable main
```

После этого обновление — обычный `apt upgrade`. Команду `--update` можно не трогать: она сама уступит apt.

Готовые хостинги, если не хочется крутить reprepro: [Cloudsmith](https://cloudsmith.io), [Gemfury](https://gemfury.com), [Packagecloud](https://packagecloud.io).

## Официальный Debian/Ubuntu

Имеет смысл только если пакет нужен тысячам чужих машин. Тогда: ITP на bugs.debian.org, пакет по Policy, спонсор из Debian. Это месяцы и не про мониторинг своего флота.

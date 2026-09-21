# Linux file-manager integration

`cloudreve-share` opens the share dialog for a local path via the
`cloudreve://share/<path>` deep link — the same dialog the Windows Explorer
context menu opens. Works whether the app is running or not.

## Helper

```sh
cp cloudreve-share ~/.local/bin/
chmod +x ~/.local/bin/cloudreve-share
cloudreve-share ~/Cloudreve/report.pdf
```

## GNOME / Nautilus

```sh
mkdir -p ~/.local/share/nautilus/scripts
cp "nautilus/scripts/Cloudreve Share" ~/.local/share/nautilus/scripts/
chmod +x ~/.local/share/nautilus/scripts/"Cloudreve Share"
```

Right-click a file → **Scripts** → **Cloudreve Share**.

## KDE / Dolphin

```sh
mkdir -p ~/.local/share/kio/servicemenus   # Dolphin ≥ 22
cp kde/cloudreve-share.desktop ~/.local/share/kio/servicemenus/
```

Right-click a file → **Cloudreve** → **Share link…**. Restart Dolphin if the
entry does not appear.

Requires `python3` and `xdg-open` (both standard on desktop Linux).

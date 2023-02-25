# Installation

You can install stitchmd from [pre-built binaries](#binary-installation)
or [from source](#install-from-source).

## Binary installation

Pre-built binaries of stitchmd are available for different platforms
over a few different mediums.

### Homebrew

If you use **Homebrew** on macOS or Linux,
run the following command to install stitchmd:

```bash
brew install abhinav/tap/stitchmd
```

### ArchLinux

If you use **ArchLinux**,
install stitchmd from [AUR](https://aur.archlinux.org/)
using the [stitchmd-bin](https://aur.archlinux.org/packages/stitchmd-bin/)
package.

```bash
git clone https://aur.archlinux.org/stitchmd-bin.git
cd stitchmd-bin
makepkg -si
```

If you use an AUR helper like [yay](https://github.com/Jguer/yay),
run the following command instead:

```go
yay -S stitchmd-bin
```

### GitHub Releases

For **other platforms**, download a pre-built binary from the
[Releases page](https://github.com/abhinav/stitchmd/releases)
and place it on your `$PATH`.

## Install from source

To install stitchmd from source, [install Go >= 1.20](https://go.dev/dl/)
and run:

```bash
go install go.abhg.dev/stitchmd@latest
```

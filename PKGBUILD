pkgname=dbf-reader
pkgver=1.0.0
pkgrel=1
pkgdesc="Convert DBF files to CSV"
arch=('x86_64')
url="local"
license=('MIT')
depends=()
makedepends=('go')
source=()
sha256sums=()

build() {
  cd "$srcdir"
  # Adjust this path if your source isn't in the same directory as PKGBUILD.
  # Common pattern is to put the source alongside PKGBUILD or use a git source.
  go build -o dbf-reader /home/plutonicvoid/code/DBFreader
}

package() {
  install -Dm755 "$srcdir/dbf-reader" "$pkgdir/usr/bin/dbf-reader"
}
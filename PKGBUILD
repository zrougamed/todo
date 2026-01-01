# Maintainer: nirabyte 
pkgname=todo-bin
pkgver=1.0
pkgrel=1
pkgdesc="Interactive keyboard-driven TUI todo list manager with 10 color themes, 30 completion animations, timer notifications, and persistent JSON storage"
arch=('x86_64')  
url="https://github.com/nirabyte/todo"
license=('unknown') 
provides=('todo')
conflicts=('todo')
source=("todo::$url/releases/download/v$pkgver/todo")
sha256sums=('105a5221033e3b7ecad62b07fe938f1cd344c731a0291a6545edbab18a5cb9f0')

package() {
  install -Dm755 "todo" "$pkgdir/usr/bin/todo"
}
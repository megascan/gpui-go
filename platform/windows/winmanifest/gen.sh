#!/bin/sh
set -eu
cd "$(dirname "$0")"
llvm-windres --target=x86_64-w64-mingw32 -O coff gpui.rc rsrc_windows_amd64.syso
llvm-windres --target=aarch64-w64-mingw32 -O coff gpui.rc rsrc_windows_arm64.syso
llvm-windres --target=i686-w64-mingw32 -O coff gpui.rc rsrc_windows_386.syso

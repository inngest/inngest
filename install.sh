#!/bin/sh

# Hi!  We're glad that you're inspecting this script before running - good idea!
#
# You're looking at a bash script that:
#
# - Checks for latest inngest CLI releases
# - Downloads the correct release artifact for your system
# - Verifies the checksum
# - Untars/unzips it
#
# There's no need for sudo.  We don't do anything else :)
#
# Reach out to us (founders@inngest.com) if you have any questions.

set -e

binname="inngest"
reponame="inngest"
base="https://github.com/inngest"

cat /dev/null <<EOF
------------------------------------------------------------------------
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
------------------------------------------------------------------------
EOF
is_command() {
  command -v "$1" >/dev/null
}
echoerr() {
  echo "$@" 1>&2
}
log_prefix() {
  echo "inngestcl:"
}
_logp=7
log_set_priority() {
  _logp="$1"
}
log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}
log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}
log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}
log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}
log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}
log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}
uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}
uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo ${arch}
}

untar() {
  tarball=$1
  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ;;
    *.tar) tar --no-same-owner -xf "${tarball}" ;;
    *.zip) unzip "${tarball}" ;;
    *)
      log_err "untar unknown archive format for ${tarball}"
      return 1
      ;;
  esac
}
http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    curl -fsSL -o "$local_file" "$source_url"
  else
    curl -fsSL -H "$header" -o "$local_file" "$source_url"
  fi
}

http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    wget -q -O "$local_file" "$source_url"
  else
    wget -q --header "$header" -O "$local_file" "$source_url"
  fi
}

http_download() {
  log_debug "http $2"
  if is_command curl; then
    http_download_curl "$@"
    return
  elif is_command wget; then
    http_download_wget "$@"
    return
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}

http_copy() {
  tmp=$(mktemp)
  http_download "${tmp}" "$1" "$2" || return 1
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}

github_release() {
  owner_repo=$1
  version=$2
  test -z "$version" && version="latest"
  giturl="https://github.com/${owner_repo}/releases/${version}"
  json=$(http_copy "$giturl" "Accept:application/json")
  test -z "$json" && return 1
  version=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
  test -z "$version" && return 1
  echo "$version"
}
hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl -dst openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f a
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}
hash_sha256_verify() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "hash_sha256_verify checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  want=$(grep "${BASENAME}" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
  if [ -z "$want" ]; then
    log_err "hash_sha256_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_sha256_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

github_api() {
  local_file=$1
  source_url=$2
  header=""
  case "$source_url" in
  https://api.github.com*)
     test -z "$GITHUB_TOKEN" || header="Authorization: token $GITHUB_TOKEN"
     ;;
  esac
  http_download "$local_file" "$source_url" "$header"
}

github_last_release() {
  owner_repo=$1
  giturl="https://api.github.com/repos/${owner_repo}/releases/latest"
  html=$(github_api - "$giturl")
  version=$(echo "$html" | grep -m 1 "\"tag_name\":" | cut -f4 -d'"')
  test -z "$version" && return 1
  echo "$version"
}

cat /dev/null <<EOF
------------------------------------------------------------------------
End of functions from https://github.com/client9/shlib
------------------------------------------------------------------------
EOF

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}

uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="x86_64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
  esac
  echo "$arch"
}

if [ -z "${VERSION}" ]; then
  log_info "checking GitHub for latest version"
  VERSION=$(github_last_release "inngest/$reponame")
fi

# if version starts with 'v', remove it
VERSION=${VERSION#v}

base_url() {
    os="$(uname_os)"
    arch="$(uname_arch)"
    url="${base}/${reponame}/releases/download/v${VERSION}"
    echo "$url"
}

tarball() {
    os="$(uname_os)"
    arch="$(uname_arch)"
    name="${reponame}_${VERSION}_${os}_${arch}"
    if [ "$os" = "windows" ]; then
        name="${name}.zip"
    else
        name="${name}.tar.gz"
    fi
    echo "$name"
}

execute() {
    base_url="$(base_url)"
    tarball="$(tarball)"
    tarball_url="${base_url}/${tarball}"
    checksum="checksums.txt"
    checksum_url="${base_url}/${checksum}"
    bin_dir="./"
    binexe=$binname

    tmpdir=$(mktemp -d)
    log_debug "downloading files into ${tmpdir}"
    http_download "${tmpdir}/${tarball}" "${tarball_url}"
    http_download "${tmpdir}/${checksum}" "${checksum_url}"

    hash_sha256_verify "${tmpdir}/${tarball}" "${tmpdir}/${checksum}"
    srcdir="${tmpdir}"
    (cd "${tmpdir}" && untar "${tarball}")
    test ! -d "${bin_dir}" && install -d "${bin_dir}"
    install "${srcdir}/${binexe}" "${bin_dir}"
    log_info "installed ${bin_dir}${binexe}"
    rm -rf "${tmpdir}"

    printf "\n${binexe} has been installed into ${bin_dir}${binexe}.  To place ${binexe} into your path run:\n"
    printf "\tsudo mv ${bin_dir}${binexe} /usr/local/bin/${binexe}\n"
}

execute

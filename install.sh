#!/usr/bin/env sh
# aiswitch installer — downloads a prebuilt binary from GitHub Releases.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/higordiego/ai-switch/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/higordiego/ai-switch/main/install.sh | VERSION=v0.1.2 sh
#   curl -fsSL https://raw.githubusercontent.com/higordiego/ai-switch/main/install.sh | INSTALL_DIR=/usr/local/bin sh
#
# No Go toolchain required.

set -eu

REPO="${REPO:-higordiego/ai-switch}"
BINARY_NAME="${BINARY_NAME:-aiswitch}"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${VERSION:-}"
GITHUB_API="${GITHUB_API:-https://api.github.com}"
GITHUB_URL="${GITHUB_URL:-https://github.com}"

tmp_dir=""
cleanup() {
  if [ -n "${tmp_dir}" ] && [ -d "${tmp_dir}" ]; then
    rm -rf "${tmp_dir}"
  fi
}
trap cleanup EXIT INT TERM

say() { printf '%s\n' "$*"; }
err() { printf 'aiswitch-install: %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "comando obrigatorio ausente: $1"
}

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux) printf 'linux\n' ;;
    darwin) printf 'darwin\n' ;;
    *) err "sistema nao suportado: $(uname -s) (use macOS ou Linux)" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *) err "arquitetura nao suportada: $arch" ;;
  esac
}

http_get() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 1 -o "$out" "$url" \
      || err "falha ao baixar: $url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$out" "$url" || err "falha ao baixar: $url"
  else
    err "instale curl ou wget"
  fi
}

http_get_stdout() {
  url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 1 "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O - "$url"
  else
    err "instale curl ou wget"
  fi
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    case "$VERSION" in
      v*) printf '%s\n' "$VERSION" ;;
      *) printf 'v%s\n' "$VERSION" ;;
    esac
    return
  fi
  # Latest release tag via GitHub API (no jq required).
  json="$(http_get_stdout "${GITHUB_API}/repos/${REPO}/releases/latest")" \
    || err "nao foi possivel consultar a release latest de ${REPO}"
  tag="$(printf '%s' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$tag" ] || err "tag_name nao encontrado na API de releases"
  printf '%s\n' "$tag"
}

checksum_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    err "precisa de sha256sum ou shasum para verificar o download"
  fi
}

in_path() {
  dir="$1"
  printf '%s' "$PATH" | tr ':' '\n' | grep -Fx "$dir" >/dev/null 2>&1
}

main() {
  need_cmd uname
  need_cmd mktemp
  need_cmd mkdir
  need_cmd chmod
  need_cmd mv
  need_cmd sed
  need_cmd awk
  need_cmd head
  need_cmd tr

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  asset="${BINARY_NAME}_${version}_${os}_${arch}"
  base="${GITHUB_URL}/${REPO}/releases/download/${version}"
  bin_url="${base}/${asset}"
  sum_url="${base}/${asset}.sha256"

  say "aiswitch installer"
  say "  repo:    ${REPO}"
  say "  version: ${version}"
  say "  target:  ${os}/${arch}"
  say "  dest:    ${INSTALL_DIR}/${BINARY_NAME}"

  tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/aiswitch-install.XXXXXX")"
  bin_path="${tmp_dir}/${asset}"
  sum_path="${tmp_dir}/${asset}.sha256"

  say "baixando ${bin_url}"
  http_get "$bin_url" "$bin_path"
  say "baixando ${sum_url}"
  http_get "$sum_url" "$sum_path"

  expected="$(awk '{print $1}' "$sum_path" | head -n 1)"
  [ -n "$expected" ] || err "checksum vazio em ${asset}.sha256"
  actual="$(checksum_file "$bin_path")"
  if [ "$expected" != "$actual" ]; then
    err "checksum SHA-256 invalido (esperado ${expected}, obtido ${actual})"
  fi
  say "checksum OK"

  mkdir -p "$INSTALL_DIR"
  chmod 755 "$bin_path"
  # Atomic replace when possible.
  mv -f "$bin_path" "${INSTALL_DIR}/${BINARY_NAME}"
  chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"

  say ""
  say "instalado: ${INSTALL_DIR}/${BINARY_NAME}"
  if "${INSTALL_DIR}/${BINARY_NAME}" version >/dev/null 2>&1; then
    say "versao:    $("${INSTALL_DIR}/${BINARY_NAME}" version)"
  fi

  if ! in_path "$INSTALL_DIR"; then
    say ""
    say "adicione ao PATH (zsh):"
    say "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
    say "adicione ao PATH (bash):"
    say "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
  fi

  say ""
  say "proximos passos:"
  say "  aiswitch doctor"
  say "  aiswitch                 # TUI"
  say "  eval \"\$(aiswitch shell-init zsh)\" && aiswitch use <perfil>"
  say ""
  say "feito por Higor Diego"
}

main "$@"

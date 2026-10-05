#!/bin/sh
# Run the selected Rust project's native baseline without changing its toolchain.
set -eu

usage() {
    printf 'Usage: sh %s [rust-project-directory|Cargo.toml] [--skip-fmt] [-p package ...]\n' "$0" >&2
    exit 2
}

project=.
if [ "$#" -gt 0 ]; then
    case $1 in
        -*) ;;
        *) project=$1; shift ;;
    esac
fi
check_fmt=true
if [ "$#" -gt 0 ] && [ "$1" = "--skip-fmt" ]; then
    check_fmt=false
    shift
fi
need_package=false
for argument
do
    if [ "$need_package" = true ]; then
        case $argument in ""|-*) usage ;; esac
        need_package=false
    else
        case $argument in
            -p|--package) need_package=true ;;
            *) usage ;;
        esac
    fi
done
[ "$need_package" = false ] || usage
[ -n "$project" ] || usage
case $project in
    /*) ;;
    *) project=./$project ;;
esac
if [ -f "$project" ]; then
    manifest_name=${project##*/}
    project=${project%/*}
else
    manifest_name=Cargo.toml
fi
if ! cd -P "$project"; then
    printf 'check-rust: cannot enter project directory: %s\n' "$project" >&2
    exit 1
fi
if ! command -v cargo >/dev/null 2>&1; then
    printf 'check-rust: cargo is required on PATH (with rustfmt and Clippy for the selected toolchain)\n' >&2
    exit 1
fi

if manifest=$(cargo locate-project --workspace --manifest-path "$manifest_name" --message-format plain); then
    lockfile=${manifest%/*}/Cargo.lock
else
    status=$?
    printf 'check-rust: cannot locate a Rust workspace in the selected directory\n' >&2
    exit "$status"
fi
if [ ! -f "$lockfile" ]; then
    printf 'check-rust: intentional workspace lockfile required: %s\n' "$lockfile" >&2
    printf 'Create and review it in the Rust project before running this baseline.\n' >&2
    exit 1
fi

run() {
    printf '\n+ %s\n' "$*"
    if "$@"; then
        return 0
    else
        status=$?
        printf 'check-rust: command failed (%s): %s\n' "$status" "$*" >&2
        exit "$status"
    fi
}

# Select linkable/testable packages explicitly for workspaces containing PyO3 cdylibs.
if [ "$#" -eq 0 ]; then
    set -- --workspace
fi
if [ "$check_fmt" = true ]; then
    run cargo fmt --manifest-path "$manifest_name" --all -- --check
fi
run cargo check --manifest-path "$manifest_name" "$@" --all-targets --locked
run cargo test --manifest-path "$manifest_name" "$@" --locked
run cargo doc --manifest-path "$manifest_name" "$@" --no-deps --locked
# Portable CI warning policy; do not enable extra Clippy lint groups.
run cargo clippy --manifest-path "$manifest_name" "$@" --all-targets --locked -- -D warnings

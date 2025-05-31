#!/usr/bin/env python3
import argparse
import hashlib
import re
import sys
from collections import Counter
from pathlib import Path

import requests

# Pre‑compile regexes for performance
# Matches <span> or <div> tags, and ensure the closing tag matches the opening one (span or div).
ICON_PATTERN = re.compile(
    r'<(span|div)[^>]*\bclass="[^"]*\bmaterial-symbols-rounded(?:-fill)?(?:-\d+)?\b[^"]*"[^>]*>\s*(?P<icon>[\w_]+)\s*</\1>'
)
CSS_URL_PATTERN = re.compile(r"src:\s*url\((https://[^)]+)\)")


def scan_icons(roots: list[Path]):
    """
    Walk through all .jet.html and .templ files under each directory in `roots`,
    extract Material Symbols icons, and classify them as filled vs unfilled.

    Returns:
      (unfilled_set, filled_set, icon_counter, file_count, instance_count)
    """
    unfilled = set()
    filled = set()
    counter = Counter()
    instance_count = 0

    files_to_scan = []
    for root_dir in roots:
        # Scan for both .jet.html and .templ files
        for pattern in ["*.jet.html", "*.templ"]:
            files_to_scan.extend(root_dir.rglob(pattern))

    # Process unique files, sorted for deterministic behavior
    unique_files = sorted(list(set(files_to_scan)))
    file_count = len(unique_files)

    for file_path in unique_files:
        try:
            text = file_path.read_text(encoding="utf-8")
        except Exception as e:
            print(f"Warning: cannot read {file_path}: {e}")
            continue

        # Find all icon <span> or <div> tags
        for match in ICON_PATTERN.finditer(text):
            name = match.group("icon")
            # Check for the presence of "material-symbols-rounded-fill" in the entire matched tag's class attribute string
            is_filled = "material-symbols-rounded-fill" in match.group(0)

            counter[name] += 1
            instance_count += 1
            (filled if is_filled else unfilled).add(name)

    return unfilled, filled, counter, file_count, instance_count


def fetch_google_css(icon_names: list[str], fill: bool = False) -> str:
    """
    Fetch the Google Fonts CSS snippet for a list of icons.
    If `fill` is True, fetch the filled variant.
    """
    if not icon_names:
        return ""

    fill_flag = "1" if fill else "0"
    # Sort icon names to ensure consistent URL generation, which might help with caching or diffing.
    names_param = ",".join(sorted(list(icon_names)))
    url = (
        "https://fonts.googleapis.com/css2?family=Material+Symbols+Rounded:"
        f"opsz,FILL@20..48,{fill_flag}&icon_names={names_param}&display=block"
    )
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
        "Accept": "text/css",
    }
    resp = requests.get(url, headers=headers)
    resp.raise_for_status()
    return resp.text


def extract_font_url(css: str) -> str:
    """
    Extract the first https:// URL from a src:url(...) declaration in the CSS.
    """
    match = CSS_URL_PATTERN.search(css)
    return match.group(1) if match else ""


def download_if_updated(url: str, dest: Path) -> bool:
    """
    Download the binary at `url` into `dest` only if it differs from an existing file.
    Returns True if we created/updated the file.
    """
    dest.parent.mkdir(parents=True, exist_ok=True)
    resp = requests.get(url)
    resp.raise_for_status()
    new_hash = hashlib.md5(resp.content).hexdigest()

    if dest.exists():
        old_hash = hashlib.md5(dest.read_bytes()).hexdigest()
        if old_hash == new_hash:
            print(f"{dest.name} is already up to date.")
            return False

    dest.write_bytes(resp.content)
    print(f"Downloaded new font file: {dest}")
    return True


def bump_css_version(css_file: Path, fill: bool = False) -> bool:
    """
    In `css_file`, find the woff2?v=N fragment for the rounded (or -fill) font
    and bump N by +1. Returns True if a bump occurred.
    """
    if not css_file.exists():
        print(f"Warning: CSS file {css_file} not found. Cannot bump version.")
        return False

    text = css_file.read_text(encoding="utf-8")
    suffix = "-fill" if fill else ""
    pattern = re.compile(rf"(material-symbols-rounded{suffix}\.woff2\?v=)(\d+)")
    match = pattern.search(text)
    if not match:
        print(
            f"No version tag to bump for {'filled' if fill else 'unfilled'} icons in {css_file}."
        )
        return False

    prefix, version = match.group(1), int(match.group(2))
    new_text = pattern.sub(f"{prefix}{version+1}", text)
    css_file.write_text(new_text, encoding="utf-8")
    print(
        f"Bumped CSS version in {css_file} for {'filled' if fill else 'unfilled'} icons: {version} → {version+1}"
    )
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Scan for Material Symbols icons and optionally update fonts."
    )

    default_scan_paths = [Path("assets")]
    parser.add_argument(
        "directories",
        type=Path,
        nargs="*",
        default=default_scan_paths,
        help=(
            "Directories to scan (.jet.html and .templ files). "
            f"If not specified, defaults to: {', '.join(map(str, default_scan_paths))}."
        ),
        metavar="DIRECTORY",
    )
    parser.add_argument(
        "--update-fonts",
        action="store_true",
        help="Fetch and install updated font files from Google.",
    )
    parser.add_argument(
        "--css-path",
        type=Path,
        default=Path("assets/css/tailwind-style_source.css"),
        help="CSS file in which to bump the font version.",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Print per-icon counts.",
    )

    # If the script is called with no arguments at all, print help and exit.
    if len(sys.argv) == 1:
        parser.print_help(sys.stderr)
        sys.exit(1)

    args = parser.parse_args()

    # 1) Scan HTML and Templ files
    # args.directories will be a list of Path objects.
    unfilled, filled, counts, files_scanned, total_icons = scan_icons(args.directories)

    # 2) Report summary
    print("=" * 40)
    print(
        f"Scanned {files_scanned} files from {len(args.directories)} root path(s), found {total_icons} icon instances."
    )
    print(f"Unfilled icons: {len(unfilled)}, Filled icons: {len(filled)}")
    print("=" * 40)

    if unfilled:
        print("Unfilled:", ", ".join(sorted(unfilled)))
    if filled:
        print("Filled:  ", ", ".join(sorted(filled)))

    if args.verbose and counts:
        print("\nDetailed icon counts:")
        for name, cnt in sorted(counts.items()):
            print(f"  {name}: {cnt}")

    # 3) Optionally fetch & install updated font files
    if args.update_fonts:
        fonts_dir = Path("assets/fonts")
        # Ensure the target CSS file exists before attempting operations on it
        if not args.css_path.exists() and (
            unfilled or filled
        ):  # Only warn if there are icons to update
            print(
                f"Warning: CSS file for version bumping ({args.css_path}) not found. Will skip version bumping."
            )

        for icon_set, is_filled in ((unfilled, False), (filled, True)):
            if not icon_set:
                continue

            # Fetch CSS, extract woff2 URL, download if changed, bump CSS version
            css_content = fetch_google_css(list(icon_set), fill=is_filled)
            if not css_content:
                print(
                    f"No CSS content fetched for {'filled' if is_filled else 'unfilled'} icons."
                )
                continue

            url = extract_font_url(css_content)
            if not url:
                print(
                    f"Failed to find font URL for {'filled' if is_filled else 'unfilled'}."
                )
                continue

            font_file_name = (
                f"material-symbols-rounded{'-fill' if is_filled else ''}.woff2"
            )
            dest = fonts_dir / font_file_name
            if download_if_updated(url, dest):
                if args.css_path.exists():
                    bump_css_version(args.css_path, fill=is_filled)
                else:
                    # Warning already printed if css_path doesn't exist globally.
                    # Specific warning if only one font type has this issue (though less likely with global check)
                    print(
                        f"Skipped CSS version bump for {font_file_name} because {args.css_path} was not found."
                    )


if __name__ == "__main__":
    main()

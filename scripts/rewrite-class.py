#!/usr/bin/env python3

import argparse
import os
import re
import sys

def find_svg_files(icons_dir):
    """Finds all SVG files in the specified directory."""
    svg_files = []
    for filename in os.listdir(icons_dir):
        if filename.endswith(".svg"):
            svg_files.append(os.path.join(icons_dir, filename))
    return svg_files

def find_template_files(views_dir):
    """Finds all .jet.html files recursively in the specified directory."""
    template_files = []
    for root, _, files in os.walk(views_dir):
        for filename in files:
            if filename.endswith(".jet.html"):
                template_files.append(os.path.join(root, filename))
    return template_files

def process_svg_files(svg_files):
    """
    Extracts class attributes from SVG files, removes them, and returns a map.
    """
    icon_classes = {}
    # Regex to find the class attribute within the opening <svg> tag
    # Handles variations in spacing and attribute order before 'class'
    # Captures: 1: part before class, 2: class value, 3: part after class
    svg_class_re = re.compile(r'(<\s*svg[^>]*?)\s+class="([^"]+)"([^>]*>)', re.IGNORECASE | re.DOTALL)

    print("Processing SVG files...")
    for svg_path in svg_files:
        try:
            icon_id = os.path.splitext(os.path.basename(svg_path))[0]
            with open(svg_path, 'r', encoding='utf-8') as f:
                content = f.read()

            match = svg_class_re.search(content)
            if match:
                class_value = match.group(2).strip()
                if class_value: # Only store if class is not empty
                    icon_classes[icon_id] = class_value
                    print(f"  Found class '{class_value}' in {os.path.basename(svg_path)}")

                    # Reconstruct the opening tag without the class attribute
                    # Ensure we only replace the first match (the opening tag)
                    original_tag = match.group(0)
                    new_tag = match.group(1) + match.group(3)
                    new_content = content.replace(original_tag, new_tag, 1)

                    if new_content != content:
                        with open(svg_path, 'w', encoding='utf-8') as f:
                            f.write(new_content)
                        print(f"  Removed class attribute from {os.path.basename(svg_path)}")
                    else:
                         print(f"  Warning: Failed to remove class attribute cleanly from {os.path.basename(svg_path)}. Manual check needed.", file=sys.stderr)

            else:
                print(f"  No class attribute found in {os.path.basename(svg_path)}")

        except Exception as e:
            print(f"Error processing SVG file {svg_path}: {e}", file=sys.stderr)

    return icon_classes

def process_template_files(template_files, icon_classes):
    """
    Updates icon calls in template files to include the class attribute.
    """
    # Regex to find {{ raw: icon("icon_id") }} calls with only one argument
    icon_call_re = re.compile(r'icon\("([^"]+)"\s*\)')

    print("\nProcessing template files...")
    for template_path in template_files:
        try:
            with open(template_path, 'r', encoding='utf-8') as f:
                content = f.read()

            original_content = content
            modified = False

            # Use a function with re.sub for safe replacement
            def replace_icon_call(match):
                nonlocal modified
                icon_id = match.group(1)
                class_string = icon_classes.get(icon_id)

                if class_string:
                    # Escape potential special characters in class_string if needed,
                    # but for typical CSS classes it should be fine.
                    replacement = f'icon("{icon_id}", "{class_string}")'
                    if match.group(0) != replacement:
                         modified = True
                         print(f"  Updating icon call for '{icon_id}' in {os.path.basename(template_path)}")
                         return replacement
                    else: # Already has the correct class or some other issue
                        return match.group(0)
                else:
                    # No class found for this icon, or icon call already has >1 args
                    return match.group(0)

            content = icon_call_re.sub(replace_icon_call, content)

            if modified:
                with open(template_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"  Saved changes to {template_path}")
            # else:
            #     print(f"  No changes needed for {template_path}")


        except Exception as e:
            print(f"Error processing template file {template_path}: {e}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(description="""
        Rewrites SVG class attributes and updates Jet template icon calls.
        1. Finds class="..." in assets/icons/*.svg files.
        2. Removes the class attribute from the SVG file.
        3. Stores a map of icon_id -> class_string.
        4. Finds {{ raw: icon("icon_id") }} in assets/views/**/*.jet.html files.
        5. Updates the call to {{ raw: icon("icon_id", "class_string") }} if a class was found.
    """)
    parser.add_argument(
        "root_dir",
        nargs="?",
        default=".",
        help="Root directory of the project (default: current directory). Should contain assets/icons and assets/views.",
    )
    args = parser.parse_args()

    root_dir = os.path.abspath(args.root_dir)
    icons_dir = os.path.join(root_dir, "assets", "icons")
    views_dir = os.path.join(root_dir, "assets", "views")

    if not os.path.isdir(icons_dir):
        print(f"Error: Icons directory not found at {icons_dir}", file=sys.stderr)
        sys.exit(1)
    if not os.path.isdir(views_dir):
        print(f"Error: Views directory not found at {views_dir}", file=sys.stderr)
        sys.exit(1)

    svg_files = find_svg_files(icons_dir)
    if not svg_files:
        print(f"Warning: No SVG files found in {icons_dir}", file=sys.stderr)
        # Don't exit, maybe user only wants template processing if classes were previously extracted?
        # Or maybe exit? Let's proceed but the map will be empty.

    icon_classes = process_svg_files(svg_files)

    if not icon_classes:
         print("\nWarning: No classes were extracted from SVG files. Template processing might not make changes.", file=sys.stderr)
         # Decide if we should exit here? Let's continue for now.
         # sys.exit(1)


    template_files = find_template_files(views_dir)
    if not template_files:
        print(f"Warning: No *.jet.html template files found in {views_dir}", file=sys.stderr)
        sys.exit(0) # Exit cleanly if no templates found

    process_template_files(template_files, icon_classes)

    print("\nScript finished.")

if __name__ == "__main__":
    main()

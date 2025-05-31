#!/usr/bin/env fish

# Iterate over all SVG files in the assets/icons directory
for icon in assets/icons/*.svg
    # Extract the filename without the path and extension
    set icon_name (basename "$icon" .svg)

    # Check if the length of the icon name is greater than 20 characters
    if test (string length $icon_name) -gt 20
        echo $icon
        set width (rg --only-matching --regexp 'width="\d+' < $icon)
        magick -- $icon /tmp/a.png
        kitty +icat /tmp/a.png

        read -P $width" new name?" newname

        set trimmed_newname (string trim $newname)
        if test -n "$trimmed_newname"
            # Construct the new file path
            set new_icon_path "assets/icons/$trimmed_newname.svg"

            # Rename the file
            if mv "$icon" "$new_icon_path"
                echo "Renamed '$icon' to '$new_icon_path'"

                if command -v rg >/dev/null
                    # Use process substitution and NUL delimiters for safety with filenames
                    rg "$icon_name" --files-with-matches -0 | xargs -0 -I {} sh -c 'sed -i "s/'"$icon_name"'/'"$trimmed_newname"'/g" "$@"' -- {}
                    echo "Updated references from '$icon_name' to '$trimmed_newname' in project files."
                else
                    echo "Warning: 'rg' (ripgrep) not found. Skipping reference update."
                end
            else
                echo "Error renaming '$icon' to '$new_icon_path'."
            end
        else
            echo "Skipping rename for '$icon_name' as no new name was provided."
        end
        echo # Add a newline for better readability between icons
    end
end

echo "Finished processing icons."

#!/bin/sh
set -o errexit
# not posix compliant and breaks docker builds
# set -o pipefail

# Variables
BINARY_NAME="pixivfe"
GOOS=${GOOS:-$(go env GOOS)}
GOARCH=${GOARCH:-$(go env GOARCH)}
# GIT_COMMIT_DATE=$(git show -s --format=%cd --date=format:"%Y.%m.%d")
GIT_COMMIT_DATE="2025.05.31"
# GIT_COMMIT_HASH=$(git rev-parse --short HEAD)
GIT_COMMIT_HASH="a15ff4f"
REVISION="${GIT_COMMIT_DATE}-${GIT_COMMIT_HASH}"
# UNCOMMITTED_CHANGES=$(git status --porcelain)
# if [ -n "$UNCOMMITTED_CHANGES" ]; then
# 	REVISION="${REVISION}+dirty"
# fi

# Check for .env file and load it unless explicitly told not to
if [ "$1" != "--no-env-file" ] && [ -f .env ]; then
	echo ".env file found, loading environment variables"
	set -a
	. ./.env
	set +a
else
	echo "Not loading .env file"
fi

fmt() {
	echo "Formatting Go code..."
	go fmt ./...
}

check_css() {
	SOURCE_CSS_FILE="./assets/css/tailwind-style_source.css"
	CSS_FILE="./assets/css/tailwind-style.css"

	if [ ! -f "${CSS_FILE}" ]; then
		echo "${CSS_FILE} does not exist. Compiling the Tailwind CSS source now."
		if [ ! -f "${SOURCE_CSS_FILE}" ]; then
			echo "${SOURCE_CSS_FILE} does not exist. Please update your source code."
			exit 1
		fi
		tailwindcss -i "$SOURCE_CSS_FILE" -o "$CSS_FILE"
	fi
}

build() {
	echo "Building ${BINARY_NAME}..."
	go mod tidy
	if type "jq" >/dev/null >/dev/null; then
		echo "jq found. "
	fi
	check_css
	version_css
	python scripts/extract_icons.py --update-fonts
	go tool templ generate
	CGO_ENABLED=0 go build -v -ldflags="-extldflags=-static -X codeberg.org/pixivfe/pixivfe/config.revision=${REVISION}" -o "${BINARY_NAME}"
}

build_docker() {
	echo "Building ${BINARY_NAME}..."
	CGO_ENABLED=0 go build -v -ldflags="-extldflags=-static -X codeberg.org/pixivfe/pixivfe/config.revision=${REVISION}" -o "${BINARY_NAME}"
}

version_css() {
	CSS_FILE="./assets/css/tailwind-style.css"
	HTML_FILE="./assets/views/layout/twDefault.jet.html"

	echo "Computing SHA256 checksum for ${CSS_FILE}..."

	if [ ! -f "${CSS_FILE}" ]; then
		echo "Error: ${CSS_FILE} does not exist."
		exit 1
	fi

	# Compute checksum and truncate to first 8 characters for the href query parameter
	checksum=$(sha256sum "${CSS_FILE}" | awk '{print $1}' | cut -c1-8)
	echo "Truncated SHA256 checksum for ${CSS_FILE}: ${checksum}"

	# Compute the SRI hash for the integrity attribute
	echo "Computing SRI hash for ${CSS_FILE}..."
	# SRI_HASH=$(openssl dgst -sha256 -binary "${CSS_FILE}" | openssl base64 -A)
	SRI_HASH=$(sha256sum "${CSS_FILE}" | awk '{print $1}' | sed 's/\([0-9a-fA-F]\{2\}\)/\\x\1/g' | printf "%b" "$(cat)" | base64 -w 0)
	echo "Computed SRI hash: sha256-${SRI_HASH}"

	echo "Updating ${HTML_FILE} with the truncated checksum and SRI hash..."

	if [ ! -f "${HTML_FILE}" ]; then
		echo "Error: ${HTML_FILE} does not exist."
		exit 1
	fi

	# Create a backup before modifying
	cp "${HTML_FILE}" "${HTML_FILE}.bak"

	# Replace the href attribute's query parameter with the new truncated hash
	sed -i -E 's|(href="/css/tailwind-style\.css)[^"]*(" [^>]*>)|\1?hash='"${checksum}"'\2|' "${HTML_FILE}"

	# Replace the integrity attribute with the new SRI hash
	sed -i -E 's|(href="/css/tailwind-style\.css[^"]*"[^>]*integrity=")[^"]*(")|\1sha256-'"${SRI_HASH}"'\2|' "${HTML_FILE}"

	# Check if sed was successful
	if [ $? -eq 0 ]; then
		echo "Successfully updated ${HTML_FILE} with the new truncated hash and SRI hash."
		rm "${HTML_FILE}.bak"
	else
		echo "Failed to update ${HTML_FILE}. Restoring from backup."
		mv "${HTML_FILE}.bak" "${HTML_FILE}"
		exit 1
	fi
}

test() {
	echo "Running tests..."
	go test ./...
}

i18n_code() {
	echo "Extracting i18n strings from code..."
	mkdir -p i18n/locale/en
	rm -f i18n/locale/en/code.json
	go run ./i18n/codecrawler >i18n/code_strings.json
	go run ./i18n/converter <i18n/code_strings.json >i18n/locale/en/code.json
	chmod -w i18n/locale/en/code.json
}

i18n_template() {
	echo "Extracting i18n strings from templates..."
	mkdir -p i18n/locale/en
	rm -f i18n/locale/en/template.json
	go run ./i18n/templatecrawler >i18n/template_strings.json
	go run ./i18n/converter <i18n/template_strings.json >i18n/locale/en/template.json
	chmod -w i18n/locale/en/template.json
	malformed_strings=$(jq 'to_entries | .[] | select(.value | contains("\n"))' <i18n/locale/en/template.json)
	if [ -z "$malformed_strings" ]; then
		echo "No malformed strings found."
	else
		echo "Malformed strings are listed below:"
		echo "$malformed_strings"
	fi
}

i18n() {
	echo "Starting i18n extraction process..."
	i18n_code
	i18n_template
	echo "i18n extraction completed."
}

i18n_upload() {
	echo "Uploading i18n strings to Crowdin..."
	crowdin upload
}

i18n_download() {
	echo "Downloading i18n strings from Crowdin..."
	crowdin download
}

run() {
	build_docker
	echo "Running ${BINARY_NAME}..."
	./"${BINARY_NAME}"
}

watch() {
	find . -name .git -type d -prune -o -type f |
		CGO_ENABLED=0 entr -rcc go run -v -ldflags="-extldflags=-static -X codeberg.org/pixivfe/pixivfe/config.revision=${REVISION}" "."
}

clean() {
	echo "Cleaning up..."
	rm -f "${BINARY_NAME}"
}

install_pre_commit() {
	echo "Installing pre-commit hook..."
	echo '#!/bin/sh' >.git/hooks/pre-commit
	echo './build.sh test' >>.git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	echo "Pre-commit hook installed successfully."
}

help() {
	echo "Available commands:"
	echo "  all                - Run fmt, build, and test"
	echo "  fmt                - Format Go code"
	echo "  build              - Build the binary"
	echo "  build_docker       - Build the binary (for Docker, skips i18n refresh)"
	echo "  scan               - Scan Go code"
	echo "  version_css        - Compute SHA256 checksum for bootstrap-style.css and update HTML"
	echo "  test               - Run tests"
	echo "  i18n               - Extract i18n strings"
	echo "  i18n-up            - Upload strings to Crowdin"
	echo "  i18n-down          - Download strings from Crowdin"
	echo "  run                - Build and run the binary"
	echo "  watch              - Build and run the binary, restart when file changes"
	echo "  clean              - Remove the binary"
	echo "  install-pre-commit - Install testing pre-commit hook"
	echo "  help               - Show this help message"
	echo ""
	echo "Options:"
	echo "  --no-env-file - Do not load the .env file (must be the first argument if used)"
}

all() {
	fmt
	build
	test
}

# Function to handle command execution
execute_command() {
	case "$1" in
	fmt) fmt ;;
	build) build ;;
	build_docker) build_docker ;;
	version_css) version_css ;;
	test) test ;;
	i18n) i18n ;;
	i18n_code) i18n_code ;;
	i18n_template) i18n_template ;;
	i18n-up) i18n_upload ;;
	i18n_upload) i18n_upload ;;
	i18n-down) i18n_download ;;
	i18n_download) i18n_download ;;
	check_css) check_css ;;
	run) run ;;
	# watch) watch ;;
	clean) clean ;;
	# install-pre-commit) install_pre_commit ;;
	help) help ;;
	all) all ;;
	*)
		echo "Unknown command: $1"
		echo "Use 'help' to see available commands"
		exit 1
		;;
	esac
}

# Main execution
if [ $# -eq 0 ]; then
	build
elif [ "$1" = "--no-env-file" ]; then
	shift
	if [ $# -eq 0 ]; then
		build
	else
		execute_command "$@"
	fi
else
	execute_command "$@"
fi

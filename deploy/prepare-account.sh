#!/bin/sh
# shellcheck disable=SC3043
set -o errexit

# Usage: ./prepare-account.sh <PHPSESSID>

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <PHPSESSID>"
    exit 1
fi

PHPSESSID="$1"

# Common headers
USER_AGENT='Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0'
ACCEPT='application/json'
ACCEPT_LANGUAGE='en-US,en;q=0.5'
ACCEPT_ENCODING='gzip, deflate'
ORIGIN='https://www.pixiv.net'
DNT='1'
CONNECTION='keep-alive'
SEC_FETCH_DEST='empty'
SEC_FETCH_MODE='cors'
SEC_FETCH_SITE='same-origin'
PRIORITY='u=0'
PRAGMA='no-cache'
CACHE_CONTROL='no-cache'
TE='trailers'

# Function to make GET requests
make_get_request() {
    local url="$1"

    curl "$url" -X GET \
        -H "User-Agent: $USER_AGENT" \
        -H "Accept: $ACCEPT" \
        -H "Accept-Language: $ACCEPT_LANGUAGE" \
        -H "Accept-Encoding: $ACCEPT_ENCODING" \
        -H "Origin: $ORIGIN" \
        -H "DNT: $DNT" \
        -H "Connection: $CONNECTION" \
        -H "Sec-Fetch-Dest: $SEC_FETCH_DEST" \
        -H "Sec-Fetch-Mode: $SEC_FETCH_MODE" \
        -H "Sec-Fetch-Site: $SEC_FETCH_SITE" \
        -H "Pragma: $PRAGMA" \
        -H "Cache-Control: $CACHE_CONTROL" \
        -H "TE: $TE" \
        -b "PHPSESSID=$PHPSESSID" \
        --compressed \
        --silent
}

# Function to make POST requests
make_post_request() {
    local url="$1"
    local referer="$2"
    local data="$3"

    curl "$url" -X POST \
        -H "User-Agent: $USER_AGENT" \
        -H "Accept: $ACCEPT" \
        -H "Accept-Language: $ACCEPT_LANGUAGE" \
        -H "Accept-Encoding: $ACCEPT_ENCODING" \
        -H "Referer: $referer" \
        -H "Content-Type: application/json; charset=utf8" \
        -H "x-csrf-token: $CSRF_TOKEN" \
        -H "Origin: $ORIGIN" \
        -H "DNT: $DNT" \
        -H "Alt-Used: www.pixiv.net" \
        -H "Connection: $CONNECTION" \
        -H "Sec-Fetch-Dest: $SEC_FETCH_DEST" \
        -H "Sec-Fetch-Mode: $SEC_FETCH_MODE" \
        -H "Sec-Fetch-Site: $SEC_FETCH_SITE" \
        -H "Priority: $PRIORITY" \
        -H "Pragma: $PRAGMA" \
        -H "Cache-Control: $CACHE_CONTROL" \
        -H "TE: $TE" \
        -b "PHPSESSID=$PHPSESSID" \
        --compressed \
        --silent \
        --data-raw "$data"
}

# Function to extract CSRF token from response
# ref: server/routes/settings.go
extract_csrf_token() {
    local response="$1"
    echo "$response" | grep -o '"token":"[0-9a-f]\+"' | sed 's/"token":"\([0-9a-f]\+\)"/\1/'
}

# Function to handle API responses
handle_response() {
    local response="$1"
    local success_msg="$2"
    local failure_msg="$3"

    if echo "$response" | grep -q '"error":false'; then
        echo "✓ $success_msg"
    else
        echo "✗ $failure_msg"
        echo "  Error: $(echo "$response" | sed -n 's/.*"message":"\([^"]*\)".*/\1/p')"
    fi
}

# 1. Retrieve CSRF Token
TARGET_URL='https://www.pixiv.net/en/artworks/115365120'

response=$(make_get_request "$TARGET_URL")

CSRF_TOKEN=$(extract_csrf_token "$response")

if [ -z "$CSRF_TOKEN" ]; then
    echo "✗ Unable to retrieve CSRF token."
    exit 1
fi

echo "✓ CSRF token retrieved successfully: $CSRF_TOKEN"

# 2. Perform settings updates with POST requests

# 2.1 Set account country/region to Japan
response=$(make_post_request \
    'https://www.pixiv.net/ajax/settings/location' \
    'https://www.pixiv.net/settings/language-and-location' \
    '{"location":"JP"}')
handle_response "$response" "Country/region setting updated successfully." "Failed to update country/region setting."

# 2.2 Show ero-guro content (R-18G)
response=$(make_post_request \
    'https://www.pixiv.net/ajax/settings/user_x_restrict' \
    'https://www.pixiv.net/settings/viewing' \
    '{"userXRestrict":2}')
handle_response "$response" "Ero-guro content visibility setting updated successfully." "Failed to update ero-guro content visibility setting."

# 2.3 Show AI-generated work
response=$(make_post_request \
    'https://www.pixiv.net/ajax/settings/hide_ai_works' \
    'https://www.pixiv.net/settings/viewing' \
    '{"hideAiWorks":0}')
handle_response "$response" "AI-generated work visibility setting updated successfully." "Failed to update AI-generated work visibility setting."

# Internationalization

!!! warning
    **Important notice:** please don't translate server logs with `i18n.*()`. It uses the request locale, not the server one.

## Quick links to Crowdin

- [Project Dashboard](https://crowdin.com/project/pixivfe)
- [Project Settings](https://crowdin.com/project/pixivfe/settings)
- [Translation Editor](https://crowdin.com/editor/pixivfe)
- [API Key](https://crowdin.com/settings#api-key)

## Todo

- [x] Add cookie: Locale
- [x] Rewrite jet template on load to use translated strings
- [ ] Add dedicated option to set locale in settings page
- [ ] Check if any jet template strings are ignored (false negative)
- [x] Setup Crowdin
    - [x] manual upload and download
    - [x] automatic upload and download
    - [ ] automated upload and download via Woodpecker CI

## Crowdin CLI usage

Install the CLI: [See official instructions](https://crowdin.github.io/crowdin-cli/installation)

Remember to check `crowdin -V`. Should be `4.2.0` or later.

First, set up API token:

1. Go to [https://crowdin.com/settings#api-key](https://crowdin.com/settings#api-key)
2. Click the "New Token" button
3. Set permission: Projects >
    - Projects (List, Get, Create, Edit) -- Read only
    - Source files & strings (List, Get, Create, Edit) -- Read and write
    - Translations (List, Get, Create, Edit) -- Read and write
4. Copy the new token and save it somewhere

Then, try it out.

```sh
export CROWDIN_PERSONAL_TOKEN=token_here # put this somewhere in your shell config, or the `.env` file inside this repo, which will be used by ./build.sh
crowdin upload
crowdin download
```

## Crowdin Docker usage

These instructions assume `CROWDIN_PERSONAL_TOKEN` is available as an environment variable.

### Step 0: Generate i18n source files

Create `locale/en` source files (`code.json` and `template.json`):

```sh
./build.sh i18n
```

### Step 1: Set up Docker container

Create an interactive container using `crowdin/cli:latest`, mounting local files:

```sh
docker run -it --rm --name crowdin-cli \
  -e CROWDIN_PERSONAL_TOKEN=${CROWDIN_PERSONAL_TOKEN} \
  -v "./i18n/locale:/usr/crowdin-project/i18n/locale" \
  -v "./crowdin.yml:/usr/crowdin-project/crowdin.yml" \
  crowdin/cli:latest
```

### Step 2: Upload local sources to Crowdin

Update `locale/en` sources on the [Crowdin project](https://crowdin.com/project/pixivfe/sources/files):

```sh
crowdin upload --verbose
```

### Step 3: Download translations to local repository

Fetch new translations from [Crowdin](https://crowdin.com/project/pixivfe) and update non-English locales:

```sh
crowdin download --verbose
```

## Crowdin Web UI usage

Add new languages to translate in Settings > Languages.

## ./i18n/crawler -- Primer on html and jet's treatment of templates

```
<a>abc {{.Hi}}</a>
```

html: `abc {{.Hi}}` is text

jet: `<a>abc ` is text
jet: `</a>` is text

So, jet doesn't care about HTML.

And we have to care about `{* *}` comments.

## ./i18n/crawler -- Coalesce inline tags

Example: in below, the `<a>` tag should be part of the string.

```html
Log in with your Pixiv account's cookie to access features above. To learn how to obtain your cookie, please see <a href="https://pixivfe-docs.pages.dev/obtaining-pixivfe-token/">the guide on obtaining your PixivFE token</a>.
```

Tags to consider as inline: `a`

## Implementation details

This section covers implementation details for the i18n system.

The i18n system consists of the following components:

| Component | Description | Key files/functions |
|-----------|-------------|---------------------|
| Crawler | Extracts translatable strings from HTML template files | `crawler/main.go` |
| Converter | Processes crawler output to generate translation map | `converter/main.go`, `i18n.SuccintId()` |
| Locale files | Store `en` source and translations in JSON format | `i18n/locale/<lang_code>/code.json`, `i18n/locale/<lang_code>/template.json` |
| Lookup and rewrite functions | Core i18n functionality for loading translations and looking up strings | `lookup.go`, `rewrite.go` |
| Integration | Wrapper functions for automatic translation lookup | `Tr()`, `Sprintf()` |

Additional notes:

- Uses `xxHash` for string hashing when generating IDs
- Caches `strings.Replacer` objects for performance
- Supports different locales per goroutine using `routine.InheritableThreadLocal`
- Includes a Semgrep rule (`semgrep-i18n.yml`) for detecting untranslated strings

### 1. Crawler

The crawler (`crawler/main.go`) scans HTML template files to extract translatable strings.

- Uses the `html` package to parse HTML
- Traverses the DOM tree to find text nodes
- Ignores certain patterns (e.g., Jet template commands and strings included in the `IgnoreTheseStrings` variable)
- Outputs a JSON array of objects containing the message and file path

**Example output:**

```json
[
  {
    "msg": "Translatable string",
    "file": "path/to/file.html"
  }
]
```

### 2. Converter

The converter (`converter/main.go`) processes the crawler output to generate a translation map.

- Reads the crawler JSON from stdin
- Generates a unique ID for each string using `i18n.SuccintId()`
- Outputs a JSON object mapping IDs to original strings

**Example output:**

```json
{
  "path/to/file.html:uniqueId": "Translatable string"
}
```

### 3. Locale files

Translations are stored in JSON files under `i18n/locale/<lang_code>/`:

- `code.json`: Translations for strings in Go code
- `template.json`: Translations for strings in HTML templates

The base locale (English) contains the original strings, while other locales contain translated strings.

### 4. Lookup and rewrite functions

The core i18n functionality is implemented in `lookup.go` and `rewrite.go`.

#### 4.1. Lookup

`lookup.go` provides functions to load translations and look up strings:

- `Init()`: Loads all locale files into memory
- `__lookup_skip_stack_2()`: Performs the actual string lookup
- `SuccintId()`: Generates a unique ID for a string based on file path and content

#### 4.2. Rewrite

`rewrite.go` handles template rewriting:

- `Replacer()`: Returns a `strings.Replacer` for a given locale and file
- `translationPairs_inner()`: Generates replacement pairs for a locale and file

### 5. Integration

The i18n system is integrated into the application code using wrapper functions that automatically look up translations based on the current locale:

```go
func Tr(text string) string {
    return __lookup_skip_stack_2(GetLocale(), text)
}

func Sprintf(format string, a ...any) string {
    format = __lookup_skip_stack_2(GetLocale(), format)
    return fmt.Sprintf(format, a...)
}
```

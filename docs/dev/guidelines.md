# Guidelines

This file contains guidelines for code development.

!!! tip
    For general Go coding practices, please refer to the following official Go documentation:

    - [Effective Go](https://go.dev/doc/effective_go)
    - [Go Wiki: Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)

    The guidelines in this document are specific to PixivFE and should be followed in addition to the general Go best practices.

## Naming conventions

### Files

- Directory names **must** use [snake_case](https://en.wikipedia.org/wiki/Snake_case).
- File names **must** use [kebab-case](https://en.wikipedia.org/wiki/Letter_case#Kebab_case) (lowercase with hyphens).
- All characters in directory and file names **must** be lowercase.

### Code

#### Variables

- Local variables should be named using [camelCase](https://en.wikipedia.org/wiki/Camel_case) with an initial lowercase letter.
- Global variables **must** be named using [PascalCase](https://en.wikipedia.org/wiki/PascalCase) (CamelCase with an initial uppercase letter).
- Environment variables **must** be named using [SCREAMING_SNAKE_CASE](https://en.wikipedia.org/wiki/Snake_case) (all uppercase with underscores).
- Variable names should not contain any special characters.

## Adding features

- Edit the `build.sh` if you want to create scripts that help with the process of developing and/or using PixivFE.
- Don't create or add any shell files or configuration files of external programs in the root directory unless necessary.
- Add comments where necessary to explain complex logic or non-obvious code behavior.
- External libraries should only be added with prior notice.

## Repository structure

<!-- The double indentation for nested bulleted points below is required for the final doc to render properly --->

- `assets/`: Static files (images, CSS, JS) and site templates.
- `config/`: Handle configurations for the server.
    - `config.go`: Parses environment variables and passes down the configurations.
    - `proxy_list.go`: Contains the built-in third-party image proxy list.
- `core/`: Makes requests to Pixiv's server and parses information into structured data.
- `doc/`: Contains documentation for general users and developers.
- `server/`: The web server.
    - `audit/`: Audits requests made by PixivFE to external APIs.
    - `handlers/`: Contains middlewares for logging, rate limiting, error handling, etc.
    - `proxy_checker/`: The image proxy checker.
    - `request_context/`: Manages request-specific context information.
    - `routes/`: Route handling. Pages will be rendered here.
    - `session/`: Handles user's options.
    - `template/`: Core template renderer and template functions.
    - `token_manager/`: Manages and rotates API tokens with load balancing and error handling.
    - `utils/`: Other utilities.

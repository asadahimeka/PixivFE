# The current state of testing

- some playwright test
- syntax checking jet templates

## Property testing with [flyingmutant/rapid](https://github.com/flyingmutant/rapid/)

I tried doing using rapid to test templates. However, the generator is broken. The error says `reflect.Set on unexported field`.

See the [`rapid` branch](https://codeberg.org/VnPower/PixivFE/src/branch/rapid).

`interface{}` can't be used anywhere in Data_* or else rapid will complain.

# Proposed testing procedure

- Create basic GET requests to [all implemented GET routes](https://codeberg.org/VnPower/PixivFE/src/branch/v2/server/middleware/router.go) with and without parameters.
- Create basic POST requests to all implemented POST routes with different payloads.
- Perform static checks on rendered HTML pages (HTML parsing without any interactions).
- Perform dynamic checks on a live server (Interact with pages and routes using Playwright).
- Code guidelines check.

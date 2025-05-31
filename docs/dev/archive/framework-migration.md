# Migrating from gofiber to net/http -- the plan

!!! warning "Archived Content"
    This document has been archived and may contain outdated information. Please refer to the main documentation for up-to-date information.

## Tips

- To access `/:abc`, use `r.PathValue("abc")`.
- Want to associate arbitrary value with a request? Use `r.Context.Value()`.
- Don't use `r.Response`. It's `nil`.
- Only use the following HTTP status codes:
  - 303 StatusSeeOther: set method to GET
  - 307 StatusTemporaryRedirect: method and body not changed
  - 308 StatusPermanentRedirect: method and body not changed

## todo

- [x] [add caching](../features/caching.md)
- [x] add limiter (maybe it should be in nginx)
- check if everything works
  - [x] templating (this has tests)
  - [x] search, artworks, users
  - [x] novels and novel settings
  - [x] settings
  - what else?

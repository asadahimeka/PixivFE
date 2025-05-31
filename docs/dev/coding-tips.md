Some not-so-obvious tricks.

## Treat cookies as user-provided values

I think we do pretty well here.

## Jet `.Illust` vs `Illust`

Be careful when you write a .jet.html template file.

If you see error like `<variable name> not found in map[...]` when visiting a page, you need to add a dot before all data member access. e.g. from `Illust` to `.Illust`. gofiber doesn't mention this.

In Jet, variables are accessed without a dot (`CurrentPath`), while data members (the `data` parameter to `Render(...)`). are accessed with a dot in front.

There are 3 types of variables.

- Global variables. Global functions are those.
- Page variables. They are provided to Jet by the server.
- Temporary variables. You use `abc := ...` to define those.

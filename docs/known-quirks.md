---
hide:
  - navigation
---

# Known quirks

## User styles not working

### Issue

User styles, such as those applied using the [Stylus browser extension](https://add0n.com/stylus.html), may not work properly with PixivFE.

### Cause

PixivFE implements a strict [Content-Security-Policy (CSP)](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy) that prevents inline styles from being loaded.

### Solution

If you're using the Stylus browser extension, follow these steps to enable user styles:

1. Open the Stylus extension options.
2. Go to the "Advanced" section.
3. Enable the option "Circumvent CSP 'style-src' via adoptedStyleSheets".

This setting allows Stylus to bypass the CSP restriction and apply user styles correctly.

For more information, refer to [issue #1685](https://github.com/openstyles/stylus/issues/1685) on the Stylus GitHub repository.

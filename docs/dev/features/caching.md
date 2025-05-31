# Caching

I thought about caching a little bit.

## Guide for caching

The majority of the site's traffic is images, and all images on the Pixiv route can be cached forever.
  Cached in browser: pixiv headers cache images for year
  On the server, the reverse proxy (e.g. nginx) can be cached worry-free forever.

JSON requests to Pixiv can be cached for some time.
  todo: We should have a middleware handling that.

Assets are already cached properly thanks to net/http.

Every rendered page can be cached for 5 minutes or so in the browser.
However, after changing settings, the page need to be refreshed instantly.
  Cache in browser: todo: read below implement in template/render.go
  The reverse proxy shouldn't do anything about this.

The only three headers that controls caching in browser and middleboxes:

- [Age](https://httpwg.org/specs/rfc9111.html#field.age)
- [Cache-Control](https://httpwg.org/specs/rfc9111.html#field.cache-control)
- [Expires](https://httpwg.org/specs/rfc9111.html#field.expires)

## Predictive Caching

Preload header is already implemented for /artwork/ and /artwork-multi/ for the main images.

However, we can do better.

When we send an HTML page to the client, we can start fetching all the images in the page. We need a lot of correct synchronization to get it right, but it will make the browsing experience faster.

- pixivfe predictively fetch images to store in its cache.
- every proxied request goes through the cache as well.

every cache item has three possible states:

- not in cache
- pending
- cached

# Hosting an image proxy server

If you prefer not to rely on third-party image proxy servers, you can set up your own! By hosting your own proxy server, you can access images from pixiv by simply changing the [Referer](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referer) to `https://www.pixiv.net/`.

By hosting your own pixiv image proxy server, you can have more control over the caching and access to pixiv images without relying on external services.

## Caddy

The configuration uses [Caddy](https://caddyserver.com/) and acts as a reverse proxy for three different domains:

- `pximg.example.com` for `i.pximg.net`
- `static-pximg.example.com` for `s.pximg.net`
- `ugoira.example.com` for `t-hk.ugoira.com`

!!! note
    To restrict access to [same-origin](https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy) requests only, set `Cross-Origin-Resource-Policy` to `same-origin` instead of `cross-origin`.

```caddyfile
*.example.com {
	@pximg host pximg.example.com
	handle @pximg {
		reverse_proxy https://i.pximg.net {
			header_up Host "i.pximg.net"
			header_up Referer "https://www.pixiv.net/"

			header_down -Server
			header_down -Via

			header_down Cross-Origin-Embedder-Policy "require-corp"
			header_down Cross-Origin-Opener-Policy "same-origin"
			header_down Cross-Origin-Resource-Policy "cross-origin"

			transport http {
				versions 2
			}
		}
	}

	@static-pximg host static-pximg.example.com
	handle @static-pximg {
		reverse_proxy https://s.pximg.net {
			header_up Host "s.pximg.net"
			header_up Referer "https://www.pixiv.net/"

			header_down -Server
			header_down -Via

			header_down Cross-Origin-Embedder-Policy "require-corp"
			header_down Cross-Origin-Opener-Policy "same-origin"
			header_down Cross-Origin-Resource-Policy "cross-origin"

			transport http {
				versions 2
			}
		}
	}

	@ugoira host ugoira.example.com
	handle @ugoira {
		reverse_proxy https://t-hk.ugoira.com {
			header_up Host "t-hk.ugoira.com"
			header_up Referer "https://ugoira.com/" # t-hk.ugoira.com will respond with HTTP 500 without this header

			header_down -Server
			header_down -Via

			header_down Cross-Origin-Embedder-Policy "require-corp"
			header_down Cross-Origin-Opener-Policy "same-origin"
			header_down Cross-Origin-Resource-Policy "cross-origin"

			transport http {
				versions 2
			}
		}
	}
}
```

## nginx

To set up an image proxy server using [nginx](https://nginx.org/), follow these steps:

### 1. Configure the proxy cache path

Set the cache path and parameters using the [`proxy_cache_path` directive](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_cache_path) under the [`http` context](https://nginx.org/en/docs/http/ngx_http_core_module.html#http):

   ```nginx
   proxy_cache_path /path/to/cache levels=1:2 keys_zone=pximg:10m max_size=10g inactive=7d use_temp_path=off;
   ```

### 2. Set up the server block

   ```nginx
   server {
       listen 443 ssl http2;

       ssl_certificate /path/to/ssl_certificate.crt;
       ssl_certificate_key /path/to/ssl_certificate.key;

       server_name pximg.example.com; # (1)!
       access_log off;

       location / {
           proxy_cache pximg;
           proxy_pass https://i.pximg.net;
           proxy_cache_revalidate on;
           proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
           proxy_cache_lock on;
           add_header X-Cache-Status $upstream_cache_status;
           proxy_set_header Host i.pximg.net;
           proxy_set_header Referer "https://www.pixiv.net/";
           proxy_set_header User-Agent "Mozilla/5.0 (Windows NT 10.0; rv:122.0) Gecko/20100101 Firefox/122.0";

           proxy_cache_valid 200 7d;
           proxy_cache_valid 404 5m;
       }
   }
   ```

   1. Replace `pximg.example.com` with your desired domain.

## Cloudflare Workers

Alternatively, you can set up an image proxy server using [Cloudflare Workers](https://developers.cloudflare.com/workers/):

```js
addEventListener("fetch", event => {
  event.respondWith(handleRequest(event.request));
});

async function handleRequest(originalRequest) {
  try {
    let url = new URL(originalRequest.url);
    url.hostname = "i.pximg.net";

    let modifiedRequest = new Request(url, originalRequest);
    let response = await fetch(modifiedRequest, {
      headers: {
        'Referer': 'https://www.pixiv.net/',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; rv:122.0) Gecko/20100101 Firefox/122.0'
      }
    });

    if (!response.ok) {
      return new Response("Error fetching the resource.", { status: response.status });
    }

    return response;
  } catch (error) {
    console.error("Failed to fetch resource: ", error.message);

    return new Response("An error occurred while fetching the resource.", { status: 500 });
  }
}
```

## Deno

You can also set up an image proxy server using [Deno Playground](https://deno.dev) or Deno Deploy.

### Deno Playground

Simply paste the following code into the playground editor:

```ts
async function handleRequest(request: Request): Promise<Response> {
  try {
    const url = new URL(request.url);
    url.hostname = "i.pximg.net";

    const modifiedRequest = new Request(url, request);
    const response = await fetch(modifiedRequest, {
      headers: {
        'Referer': 'https://www.pixiv.net/',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.3'
      }
    });

    if (!response.ok) {
      return new Response("Error fetching the Pixiv image.", { status: response.status });
    }

    const newHeaders = new Headers(response.headers);
    //? You can replace the origin with your own PixivFE domain if you prefer a dedicated proxy.
    newHeaders.set('Access-Control-Allow-Origin', '*');

    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: newHeaders
    });
  } catch (error) {
      console.error("Failed to fetch Pixiv image: ", error);
      return new Response("An error occurred while fetching the Pixiv image.", { status: 500 });
  }
}

Deno.serve({ port: 4242 }, handleRequest);
```

### Deno Deploy

If you prefer Deno Deploy for better code maintenance, you can export the code to a GitHub repo (in project settings) or just deploy a repo that contains the `main.ts` file and set the endpoint file.

### Warning

While Deno is easy to use, there are several limitations to consider:

- Free tier is far less generous than Cloudflare Workers (1,000,000 requests and 100GB bandwidth per month). So, please take a thorough consideration if you plan to use it on a large public instance.
- It maybe requires GitHub account for deployment.

However, Deno may perform better in regions where Cloudflare Workers' speed is not so ideal.

## Using the proxy server

Once you have set up your image proxy server, you can access Pixiv images by replacing the original domain with your proxy server domain:

=== "Original URL"

    [https://i.pximg.net/img-original/img/2023/06/06/20/30/01/108783513_p0.png](https://i.pximg.net/img-original/img/2023/06/06/20/30/01/108783513_p0.png)

=== "Proxy URL"

    [https://pximg.example.com/img-original/img/2023/06/06/20/30/01/108783513_p0.png](https://pximg.example.com/img-original/img/2023/06/06/20/30/01/108783513_p0.png)

## Additional resources

- For more information, you can refer to [this article](https://pixiv.cat/reverseproxy.html) by pixiv.cat, which also serves as an image proxy server. You can try an [example image](https://i.pixiv.cat/img-original/img/2023/06/06/20/30/01/108783513_p0.png) through their proxy.

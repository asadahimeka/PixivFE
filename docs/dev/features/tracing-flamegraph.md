# Tracing

Every request to pixiv websites should go through `core/requests.go`.

Every request to pixiv websites is traced.

Every server request is traced.

## How to see flamegraph

Run PixivFE **in dev mode**, visit some pages, then visit the `/diagnostics` URI path.

## Useful for fixing Vega-Lite

- [Interactive Legend | Vega-Lite](https://vega.github.io/vega-lite/examples/interactive_legend.html)
- [Interactive legend example on Vega Editor](https://vega.github.io/editor/#/examples/vega-lite/interactive_legend)
- [Tooltip | Vega-Lite](https://vega.github.io/vega-lite/docs/tooltip.html)

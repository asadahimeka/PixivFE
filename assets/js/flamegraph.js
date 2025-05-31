const vlSpec = {
  "$schema": "https://vega.github.io/schema/vega-lite/v5.json",
  "description": "flamegraph",
  "data": {
    "url": "/diagnostics/spans.json"
  },
  "width": "container",
  "height": "container",
  "config": {
    "legend": {
      "orient": "bottom",
    }
  },
  "encoding": {
    "y": { "field": "LogLine", "type": "nominal", "axis": null },
    "x": { "field": "StartTime", "type": "temporal" },
    "x2": { "field": "EndTime", "type": "temporal" },
    "color": { "field": "RequestId", "type": "nominal" },
  },
  "mark": { "type": "bar", "tooltip": {"content": "data"} },
};

vegaEmbed('#vis', vlSpec);

---
hide:
  - navigation
---

# Instance list

<!-- **Warning: Instances listed below were deemed to have complied with [the instance rules](https://pixivfe-docs.pages.dev/instance-list/#instance-rules). Any public instance that isn't in this list should be used at your own risk.** -->

This page lists all instances of PixivFE that have been submitted to the project.

Instances are ordered from oldest to newest based on when they were submitted.

!!! tip
    To add your instance to this list, [create an issue on the PixivFE repository](https://codeberg.org/PixivFE/PixivFE/issues/new?template=.forgejo%2fissue_template%2fadd-instance.yaml) using the "Add Instance" template.

<!-- Note to page editors: The tables below only refresh their data when `mkdocs serve` is restarted, due to how the data is templated in from the CSV files.  -->

## Clearnet

These instances can be accessed through any regular web browser.

{{ read_csv('data/instances.csv') }}

!!! note "Note on Cloudflare requests on the official instance"

    The official instance uses [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/) for bot detection (but is not proxied through Cloudflare).

    Your browser will run a challenge in the background when first visiting, then occasionally about once per week.

To check the uptime history and status of these instances, visit the [PixivFE instance status page](https://stats.uptimerobot.com/FbEGewWlbX).

Instances marked as having analytics load external tracking scripts, such as [Cloudflare Web Analytics](https://developers.cloudflare.com/analytics/web-analytics/).

<!-- Human-readable list when viewing raw:

- Name: perennialte.ch (Official)
  URL: https://pixiv.perennialte.ch
  Country: SG
  Cloudflare proxy: No*
  Analytics: No

- Name: dragongoose
  URL: https://pixivfe.drgns.space
  Country: US
  Cloudflare proxy: No
  Analytics: No

- Name: ducks.party
  URL: https://pixivfe.ducks.party
  Country: NL
  Cloudflare proxy: No
  Analytics: No

- Name: darkness.services
  URL: https://pixivfe.darkness.services
  Country: US
  Cloudflare proxy: Yes
  Analytics: No

- Name: thebunny.zone
  URL: https://pixivfe.thebunny.zone
  Country: HR
  Cloudflare proxy: No
  Analytics: No -->

## Tor onion services

These instances are only accessible using [Tor](https://www.torproject.org/download/).

Hosted on the Tor network, these instances provide better privacy compared to clearnet instances, but may have slower performance due to how onion routing works.

{{ read_csv('data/instances-onion.csv') }}

<!-- Human-readable list when viewing raw:

- Name: thebunny.zone
  URL: http://pixivfe.bunny5exbgbp4sqe2h2rfq2brgrx3dhohdweonepzwfgumfyygb35wyd.onion -->

## I2P services

These instances are only accessible using [I2P network layer](https://en.wikipedia.org/wiki/I2P).

These instances provide better privacy compared to clearnet instances, but may have slower performance due to how I2P works.

{{ read_csv('data/instances-i2p.csv') }}


# Instance rules

(This instance rules was originally written by [the Invidious project](https://github.com/iv-org/documentation/blob/master/docs/instances.md))

For an instance to be added to this list, it must comply with all of the rules listed below:

1. Instances must have been up for at least a week before it can be added to this list.
2. Instances must have a stable uptime of at least 80% ([according to UptimeRobot](https://stats.uptimerobot.com/FbEGewWlbX)).
3. Instances must be served via domain name.
4. Instances must be served via HTTPS if it is a clearnet instance.
5. Instances using any DDoS Protection / MITM MUST be marked as such (e.g. Cloudflare, DDoS-Guard).
6. Instances using any type of anti-bot protection MUST be marked as such.
7. Instances must not use any type of analytics, including external scripts of any kind.
8. Any system whose goal is to modify the content served to the user (i.e web server HTML rewrite) is considered the same as modifying the source code.
9. Instances running a modified source code:
    - Must respect the [GNU AGPL](https://en.wikipedia.org/wiki/GNU_Affero_General_Public_License) by publishing their source code and stating their changes **before** they are added to the list
    - Must publish any later modification in a timely manner
10. Instances must not serve ads (sponsorship links in the banner are considered ads) NOR promote products.

**NOTE:** We reserve the right to decline any instance from being added to the list, and to remove / ban any instance breaking the aforementioned rules.

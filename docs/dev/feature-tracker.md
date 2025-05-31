# Feature tracker

This page tracks the implementation of features or redesigns in PixivFE.

Features with broader scope have their own detailed write-ups under the `dev/features` section.

## Pixivision

**Summary**: Pixivision is a service owned by Pixiv that publishes articles about various types of artwork themes...

Notes

- [Pixivision](https://www.pixivision.net/en/) is an independent service. Most of the data comes from Pixivision's website.
- Pixivision does not provide an API for data access. Pages on Pixivision seems to be static, so the HTML could be easily parsed.

Thoughts

- Write a separate module for Pixivision and integrate it into PixivFE.
- We do web scraping and HTML parsing for this one.

## Sketch

**Summary**: Sketch is a service owned by Pixiv that allow users to livestream, mostly for their drawing process.

Notes

- [Sketch](https://sketch.pixiv.net/) is an independent service. Most of the data comes from Pixivision's website.
- Sketch has a dedicated API for data access. Sketch uses the same type of authentication that Pixiv has.
- Detailed notes TBA.

Thoughts

- Thanks to the public API, pages could be build easily.
- For the streaming part, we may have to include a JavaScript library for HLS streaming.

## VRoid Hub

Service owned by Pixiv, this time for 3D humanoid models.

Example page: [スロウス 夏用 - VRoid Hub](https://hub.vroid.com/en/characters/115435665512885870/models/57790810665766270)

## Ugoira support

**Summary:** Ugoiras are Pixiv's "animated image" format.

Notes

- Ugoiras are basically a bunch of sorted images combined with a fixed delay for each of them.
- Pixiv provides one JSON endpoint for delays and filenames and one endpoint for the (ZIP) images archive.
- One has to write their own player based on things Pixiv provide.
- You can check out Pixiv's implementation on their own ugoira player [here](https://github.com/pixiv/zip_player).

Thoughts

- GIF/APNG/WEBP renderer.
- Some people want to convert ugoiras to video formats? (no idea)

## Landing page

**Summary**: PixivFE's homepage.

Notes

- Pixiv's homepage contains a lot of interesting content.
- PixivFE's backend for the landing page already implemented almost all of the data from Pixiv.
- The only thing left is to write the frontend for them.
- Detailed notes TBA.

Thoughts

- Spend some time to write some HTML/SCSS.
- Currently, you have to authenticate (login) in order to access the _full_ landing page. Can we show the _full_ page to unauthenticated users as well?

## Popular artworks

**Summary**: Pixiv has a "Sort by views" and "Sort by bookmarks" feature that is only available for premium users.

Notes

- There are some search ["hacks"](https://github.com/kokseen1/Mashiro/) that could yield relatively accurate results for popular artworks.

Ideas

- Look into repos that attempts to retrieve popular artworks
- If search "hacking" is possible, could there be more "hacks" around?

## "User discovery" page

**Summary**: Like artwork discovery, but it is for users.

Notes

- ~~Currently, we do not know if we could implement the "user follow" function into PixivFE.~~
    - ~~The development for this page has been put on hold because of it, since "following", after all, is what you want to do if you discover an user you like.~~
    - **User follow was implemented in [`#129`](https://codeberg.org/VnPower/PixivFE/pulls/129) using the API for mobile users viewing the Pixiv website (not the Pixiv app API).**

Ideas

- It is easy to implement thanks to the API.

## Search suggestions

**Summary**: Pixiv provides [an API endpoint](https://www.pixiv.net/ajax/search/suggestion?mode=all&lang=en) for search suggestions.

Notes

- The search suggestions appear when you focus on the search bar.

Ideas

- We can prefetch the search suggestions for every request on PixivFE. But this means we will have to add one request (to Pixiv's API) for each PixivFE page request. ([Caching?](features/caching.md))
- We can implement JavaScript to fetch the suggestions every time the user focuses on the search bar.
- We can create a separate page just for this.

## App API support

**Summary**: Apart of the public AJAX API, Pixiv also provides a private API, used specifically for mobile applications.

Notes

- Because you already could do almost everything through the AJAX API, there is really no point to integrate the App API.
- I added this section because there are some limitations to the public API (following,...).

Ideas

- Write more stuff when desperate.

## Novel page

## Image grid layout

## Series

## Server's PixivFE Git version/commit

**Summary**: Implementation is essentially complete. However, the removal of `.dockerignore` as a dirty workaround[^1] is unfortunate.

Ideas

- Implemented by other open-source alternative frontends, for example:
    - [Invidious](https://github.com/iv-org/invidious/blob/a021b93063f3956fc9bb3cce0fb56ea252422738/src/invidious/views/template.ecr#L117-L131)
    - [Nitter](https://github.com/zedeus/nitter/blob/b62d73dbd373f08af07c7a79efcd790d3bc1a49c/src/views/about.nim#L5-L9)

Notes

- Initial implementation by [jackyzy823](https://codeberg.org/jackyzy823) in [#104](https://codeberg.org/VnPower/PixivFE/pulls/104) ([f53e1c3e4d](https://codeberg.org/VnPower/PixivFE/pulls/104/commits/f53e1c3e4db31587ede84f5518d729fcc076dd44)) via a `REVISION` variable defined as the Git commit hash of HEAD at build time.

- The `REVISION` variable was modified to include both the commit date and hash in [`901286d98e`](https://codeberg.org/VnPower/PixivFE/commit/901286d98ec27faa7f255146ce38d7c4a87f30ed).

- A "dirty" flag is appended to the `REVISION` variable if there are uncommitted changes in [`7a9216a165`](https://codeberg.org/VnPower/PixivFE/commit/7a9216a165a10fda24666e256747420f56473f0f).

- The `.dockerignore` file was removed to prevent Docker image builds from always being flagged as "dirty" in [`436f4073ea`](https://codeberg.org/VnPower/PixivFE/commit/436f4073eaf6168946674126fe61626ba3753afd).

## Download all buttons for all containers

## Page profile

## Download button in artwork page

## Option to select default image quality in artwork page

## Pixiv App API

Besides the web API, Pixiv also has a private API made specifically for mobile applications.

There are currently no official API specifications for the private API, and the unofficial ones seem to lack several functionalities. We might have to write one.

Examples of implementations:

- [upbit/pixivpy (`aapi.py`)](https://github.com/upbit/pixivpy/blob/master/pixivpy3/aapi.py)
- [book000/pixivts (`pixiv.ts`)](https://github.com/book000/pixivts/blob/master/src/pixiv.ts)

### Goal

- Separate the Web API core and the App API core, but make both of them compatible with each other.
- Complications...

## Novels

This section tracks the implementation of features related to novels.

| Category | Feature | Status | Notes |
|----------|---------|--------|-------|
| Functions | Novel series | ✅ | Completed a while ago as of 2024-10-14, and implemented for the Bootstrap rewrite |
| UI | Furigana support | ❌ | |
| UI | Reader settings panel | ❌ | Completed a long time ago as of 2024-10-14, but the Bootstrap rewrite has a buggy/unstable implementation |
| UI | Novel page with vertical text if `body.suggestedSettings.viewMode == 1` | ❌ | Completed a long time ago as of 2024-10-14, but the Bootstrap rewrite has a buggy/unstable implementation |
| UI | Attributes | ❌ | |
| UI | Recent novels from writers | ✅ | Completed during the Bootstrap rewrite, but has incomplete data from the Pixiv API |
| UI | Page support | ❌ | |
| UI | Recommended novels | ✅ | Completed a long time ago as of 2024-10-14 |
| UI | Other works panel? | ❌ | |
| Misc | Novel ranking | ❌ | |
| Misc | Novel mode for any possible pages | ❌ | |

## Per-User Customization Options

Probably cookie-based.

### site-wide

- [ ] sidebar close on history change or not [#63](https://codeberg.org/VnPower/PixivFE/issues/63)
- [ ] navbar sticky or not

### artwork
- [ ] native AI/R15/R18/R18-G... artwork filtering
We can filter them out using values supplied by Pixiv for each artworks.

### search
- [ ] add an option to do potentially very extensive searches
Sort by bookmarks

[^1]: No pun intended.

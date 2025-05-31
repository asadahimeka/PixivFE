# Authentication for the pixiv API

This guide explains how to obtain a valid `PHPSESSID` cookie (your "token") from a signed-in pixiv account. This token allows PixivFE to access the pixiv API on your behalf.

!!! warning
    You should create an entirely new account for this to avoid account theft. And also, PixivFE will get contents **from your account.** You might not want people to know what kind of illustrations you like :P

    For now, the only page that may contain contents that is relevant to you is the discovery page. Be careful if you are using your main account.

## 1. Log in to pixiv

Log in to the pixiv account you want to use. Upon logging in, you should see the pixiv landing page. If you are already logged in, simply navigate to the landing page.

![The URL of the landing page](https://files.catbox.moe/7dbv3e.png)

## 2. Open developer tools

### For Firefox

Press `F12` to open the Firefox Developer Tools. Switch to the `Storage` tab.

![Storage tab on Firefox](https://files.catbox.moe/mra6rs.png)

### For Chrome

Press `F12` to open the Chrome Developer Tools. Switch to the `Application` tab.

![Application tab on Chrome](https://files.catbox.moe/jqpcw2.png)

## 3. Locate the cookie

### For Firefox

In the left sidebar, expand the `Cookies` section and select `www.pixiv.net`. This is where you will find your authentication cookie.

Locate the cookie with the key `PHPSESSID`. The value next to this key is your account's token.

![Cookie on Firefox](https://files.catbox.moe/zb16o8.png)

### For Chrome

In the left sidebar, find the `Storage` section. Expand the `Cookies` subsection and select `www.pixiv.net`. This is where you will find your authentication cookie.

Locate the cookie with the key `PHPSESSID`. The value next to this key is your account's token.

![PHPSESSID on Chrome-based browsers](https://files.catbox.moe/8wu9f0.png)

## 4. Set the environment variable

Copy the token value obtained in the previous step. If deploying with Docker, set it as the `PIXIVFE_TOKEN` environment variable in your configuration.

## 5. Configuring account settings (optional)

To ensure PixivFE works as intended, your pixiv account should have the following settings configured:

- Japan as account region
- R-18G content visible
- AI-generated works visible

### Automated configuration

To configure these settings automatically, we provide a `prepare-account.sh` Bash script, located in the `deploy` directory of the PixivFE repository. To use it, run the script and pass your `PHPSESSID` as an argument:

```bash
./prepare-account.sh YOUR_PHPSESSID
```

### Manual configuration

To manually configure the recommended settings:

1. Navigate to pixiv's [display settings page](https://www.pixiv.net/settings/viewing).
2. Adjust the following settings:
    - Enable "Show ero-guro content (R-18G)" if desired
    - Configure any other display preferences as needed

To verify the R-18G content visibility setting was applied:

1. Visit the [gore search endpoint](https://www.pixiv.net/ajax/search/artworks/gore).
2. Look for "R-18G" tags in the returned results.
3. If you disable the R-18G option and search again, R-18G artworks should no longer appear in the results.

## Using multiple tokens

!!! warning
    If you maintain a public PixivFE instance, it is recommended to use multiple tokens, each sourced from a different pixiv account.

pixiv enforces request limits to prevent excessive usage, and using a single account for a high volume of requests can lead to account suspension or termination.

PixivFE mitigates this risk by allowing multiple tokens to be specified in the `PIXIVFE_TOKEN` environment variable. If one account is temporarily restricted or suspended, your instance can continue using the other accounts.

## Additional notes

- The token looks like this: `123456_AaBbccDDeeFFggHHIiJjkkllmMnnooPP`
    - The underscore separates your **member ID (left side)** from a **random string (right side)**
- Logging out of pixiv will invalidate the token.
- Guides for Chrome was taken from [Nandaka's guide on logging in with cookies](https://github.com/Nandaka/PixivUtil2/wiki#pixiv-login-using-cookie).

# Current design flaws

This section documents some bad/buggy designs in PixivFE's design, both front-end and back-end.

## Low Quality Go Module: `net/url`

`url.Path` is stored decoded (no %XX). `url.Scheme` is stored without `://` (mandated by RFC). Not sure why Go does that. Felt like this is bound to cause some nasty bug on decoding and encoding.

Current proxied URLs don't have weird characters in them. Hopefully it stays this way.

Solution: Replace `net/url` with a better third-party module

~~Mitigation: replaced `url.Parse` with `urlx.Parse` from `github.com/goware/urlx`~~

**Update as of 2024-10-14:** swapping out `url.Parse` for `urlx.Parse` caused several issues and was essentially reverted over multiple bug fix commits

# sfos_orangeflex

Unofficial Sailfish OS client for **Orange Flex** (Orange Polska). Android reference: **67.9.1** (`com.orange.rn.dop`).

## Layout

| Path | What |
|---|---|
| [`apps/sfos/go/client/`](apps/sfos/go/client/) | Go library: OTP login, OAuth tokens, dashboard APIs |
| [`apps/sfos/`](apps/sfos/) | Sailfish Silica UI + c-shared C API |

## Sailfish app

```sh
sfosbuild deploy root@192.168.1.146 ./apps/sfos
```
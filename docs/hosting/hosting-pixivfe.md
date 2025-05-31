# Hosting PixivFE

PixivFE can be installed using various methods. This guide covers installation using [Docker](#docker) (recommended for production) and using a binary with a Caddy reverse proxy.

!!! note
    To function, PixivFE needs to authenticate with the pixiv API using a account's session cookie. Refer to [Authentication for the pixiv API](api-authentication.md) for detailed instructions.

## Docker

[Docker](https://www.docker.com/) lets you run containerized applications. Containers are loosely isolated environments that are lightweight and contain everything needed to run the application, so there's no need to rely on what's installed on the host.

Docker images for PixivFE are provided using our [container registry on GitLab](https://gitlab.com/pixivfe/PixivFE/container_registry/), with support for the `linux/amd64` platform.

The following Docker image tags are available:

- `latest`: The most recent stable release.
- `next`: The latest development build from the `v3` branch.
- Tagged releases (e.g., `v2.11`).

When using Docker commands, you can specify the desired tag. For example:

```bash
docker pull registry.gitlab.com/pixivfe/pixivfe:latest
docker pull registry.gitlab.com/pixivfe/pixivfe:next
docker pull registry.gitlab.com/pixivfe/pixivfe:v2.11
```

### 1. Set up the repository

Clone the PixivFE repository and navigate to the `deploy` directory:

```bash
git clone https://codeberg.org/PixivFE/PixivFE.git && cd PixivFE/deploy
```

### 2. Configure environment variables

Copy `.env.example` to `.env` and configure the variables as needed. Refer to [Configuration options](configuration-options.md) for more information.

!!! note
    Ensure you set `PIXIVFE_HOST=0.0.0.0` in the `.env` file.

    This allows PixivFE to bind to all network interfaces inside the container, which is necessary for Docker networking to function correctly.

    Any network access restrictions will be handled by Docker itself, not within PixivFE.

### 3. Start PixivFE

!!! warning
    Using Docker Compose requires the Compose plugin to be installed. Follow these [instructions on the Docker Docs](https://docs.docker.com/compose/install) on how to install it.

Run either of the following commands to start PixivFE, listening on `127.0.0.1:8282` on the host by default:

=== "Docker Compose"
    ```bash
    docker compose up -d
    ```

=== "Docker CLI"
    ```bash
    docker run -d --name pixivfe -p 127.0.0.1:8282:8282 --env-file .env registry.gitlab.com/pixivfe/pixivfe:latest
    ```

To view the container logs, run `docker logs -f pixivfe`.

## Binary

This setup uses [Caddy](https://caddyserver.com/) as the reverse proxy. Caddy is a great alternative to [NGINX](https://nginx.org/en/) because it is written in the [Go programming language](https://go.dev/), making it more lightweight and efficient. Additionally, Caddy is easy to configure, providing a simple and straightforward way to set up a reverse proxy.

### 1. Setting up the repository

Clone the PixivFE repository and navigate to the `deploy` directory:

```bash
git clone https://codeberg.org/PixivFE/PixivFE.git && cd PixivFE/deploy
```

### 2. Configure environment variables

Copy `.env.example` to `.env` and configure the variables as needed. Refer to [Configuration options](configuration-options.md) for more information.

### 3. Building and running PixivFE

PixivFE provides a shell script named `build.sh` to simplify the build and run process.

To build and run PixivFE, use the following commands:

```bash
./build.sh run
```

This will build the PixivFE binary and start it. It will be accessible at `localhost:8282`.

### 4. Deploying Caddy

[Install Caddy](https://caddyserver.com/docs/install) using your package manager.

In the PixivFE directory, create a file named `Caddyfile` with the following content:

```caddy
example.com {
  reverse_proxy localhost:8282
}
```

Replace `example.com` with your domain and `8282` with the PixivFE port if you changed it.

Run `caddy run` to start Caddy.

## Updating

To update PixivFE to the latest version, follow the steps below that are relevant to your deployment method.

### Docker

#### Docker Compose

1. Pull the latest Docker image and repository changes:
   ```bash
   docker compose pull && git pull
   ```

2. Restart the container:
   ```bash
   docker compose up -d
   ```

#### Docker CLI

1. Pull the latest Docker image and repository changes:
   ```bash
   docker pull registry.gitlab.com/pixivfe/pixivfe:latest && git pull
   ```

2. Stop and remove the existing container:
   ```bash
   docker stop pixivfe && docker rm pixivfe
   ```

3. Restart the container:
   ```bash
   docker run -d --name pixivfe -p 8282:8282 --env-file .env registry.gitlab.com/pixivfe/pixivfe:latest
   ```

### Binary

1. Pull the latest changes from the repository:
   ```bash
   git pull
   ```

2. Rebuild and start PixivFE:
   ```bash
   ./build.sh build
   ./build.sh run
   ```

## Acknowledgements

- [Keep Caddy Running](https://caddyserver.com/docs/running#keep-caddy-running)

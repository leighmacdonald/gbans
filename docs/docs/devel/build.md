# Building

Official [docker images](https://github.com/leighmacdonald/gbans/pkgs/container/gbans) are generally the
recommended way to install gbans, however if you want to build from source there is several steps you must
follow to produce a working build.

## Sentry (Optional)

Sentry is an application that handles performance monitoring and error tracking in a fairly
easy to use web interface.

We take the sentry recommended approach of splitting the backend and frontend
components of
the project into distinct sentry projects.  

If you do not set `SENTRY_DSN/SENTRY_AUTH_TOKEN` env vars when building, then
support will be effectively disabled.

### Backend Configuration

When building you should set `SENTRY_DSN=<YOUR_SENTRY_DSN>` when calling goreleaser
to embed the value into
the binary. If you prefer to instead set this at runtime you can also set the same env var when running the
gbans binary.

### Frontend Configuration

For frontend integration you can add your `SENTRY_*` values to `frontend/.env.sentry-build-plugin` or otherwise
set them when calling `just frontend`. This will embed them into the sentry plugin.

```shell
SENTRY_AUTH_TOKEN="<SENTRY_AUTH_TOKEN>"
SENTRY_URL="<SENTRY_URL>"
SENTRY_ORG="<SENTRY_ORG>"
SENRTY_PROJECT="<YOUR_PROJECT_SLUG>"
```

You must also set the values which are embedded via vite, these must start with `VITE_*` to be embedded, they will
otherwise be ignored.

```shell
VITE_BUILD_VERSION="<VITE_BUILD_VERSION>"
VITE_SENTRY_DSN="<VITE_SENTRY_DSN>"
```

## Building (Production)

We use [goreleaser](https://goreleaser.com/) for building our releases, however this assumes some things such as
having a `GITHUB_TOKEN` via running under github-actions.

```sh
goreleaser release --clean
./dist/gbans_linux_amd64_v1/gbans
```

Alternatively you can manually build the components without goreleaser.

```sh
just frontend
go build -tags release -ldflags="-s -w \
    -X 'github.com/leighmacdonald/gbans/internal/app.BuildVersion="master"' \
    -X 'github.com/leighmacdonald/gbans/internal/app.BuildCommit="master"' \
    -X 'github.com/leighmacdonald/gbans/internal/app.BuildDate=""' \
    -X 'main.builtBy=""' "
./gbans
```

Production releases will embed the frontend assets into the binary so you need to ensure that you build that first.

## Building (Development)

The development build is exactly the same except we don't want to specify the `release` tag. It's also ok to simplify the
build command to the standard:

```sh
go build
```

This build does not serve any files unlike the production build and instead assumes you are using
the `pnpm run serve` command to start the vite live-reload development server (`vite --open`).

## Creating New Release

The `release.sh` script handles automating bumping version numbers, tagging the release and running `goreleaser`.

```sh
./release.sh 0.1.2 # What version to set for the release
```

This is designed to be run from ci, like github-actions.

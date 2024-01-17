# Building

Official [docker images](https://github.com/leighmacdonald/gbans/pkgs/container/gbans) are generally the 
recommended way to install gbans, however if you want to build from source there is several steps you must 
follow to produce a working build.

## Sentry (Optional)

Sentry is an application that handles performance monitoring and error tracking in a fairly
easy to use web interface.

We take the sentry recommended approach of splitting the backend and frontend components of 
the project into distinct sentry projects.  

### Backend

After creating your backend project you can copy and paste the url shown into `gbans.yml`
under the `logging.sentry_dsn` key. No further configuration should be required.


### Frontend

After creating your backend project you can copy and paste the url shown into `gbans.yml`
under the `logging.sentry_dsn_web` key.

For frontend integration you much also create a new auth token and save it under `frontend/.env.sentry-build-plugin` along
with the `ORG` and `PROJECT` you have configured in sentry.

    SENTRY_ORG="<YOUR_ORG>"
    SENTRY_PROJECT="<YOUR_PROJECT>"
    SENTRY_AUTH_TOKEN=<YOUR_TOKEN>

You can find out more details under their [webpack docs](https://docs.sentry.io/platforms/javascript/guides/react/sourcemaps/uploading/webpack/)


## Creating New Release

The `release.sh` script handles automating bumping version numbers, tagging the release and running `goreleaser`.

    ./release.sh 0.1.2 # What version to set for the release


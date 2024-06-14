# Updating Docs

## Initialize Dependencies

If its your first time running the docs, you will need to install the dependencies first:

```shell
make docs_setup
```

## Starting a local server

To get up and running with the local development web server, run the following command:

```shell
make docs_start
```

This should open a browser tab with the current docs. If it does not open for you, the default
address is [http://localhost:3000/gbans/](http://localhost:3000/gbans/)

## Deployment

Docs for the project are currently auto deployed on every push to the master branch.

## Building

If you wish to build a production version of the docs you may use:

```shell
make docs_build
```
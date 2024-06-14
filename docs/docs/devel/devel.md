# Development Environment Setup

## Linters & Formatting

`make fmt` Format go code with go fmt

`make check` will run the go & typescript linters & static analyzers

## Updating Dependencies

`make bump_deps` Can be used to update both go and js/ts libraries.

## Start Backend

`make dev` Run the backend server. By default, listens on http://localhost:6006/. 

## Start Frontend

`make serve` Start the vite frontend development web server. This should open http://localhost:6007/ automatically.


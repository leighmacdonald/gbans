# Development Environment Setup

## Linters & Formatting

`just fmt` Format go code with go fmt

`just check` will run the go & typescript linters & static analyzers

## Updating Dependencies

`just bump_deps` Can be used to update both go and js/ts libraries.

## Start Backend

`just dev` Run the backend server. By default, listens on http://localhost:6006/. 

## Start Frontend

`just serve` Start the vite frontend development web server. This should open http://localhost:6007/ automatically.

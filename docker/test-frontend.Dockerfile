FROM node:20-alpine
LABEL maintainer="Leigh MacDonald <leigh.macdonald@gmail.com>"
RUN apk add make curl && corepack enable pnpm
WORKDIR /build

COPY Makefile .
COPY frontend/package.json frontend/package.json
COPY frontend/pnpm-lock.yaml frontend/pnpm-lock.yaml
COPY frontend frontend

RUN cd frontend && pnpm install --frozen-lockfile --strict-peer-dependencies
RUN make test-ts

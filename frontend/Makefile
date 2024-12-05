all: install build

install:
	pnpm install --frozen-lockfile

build:
	pnpm run build

watch:
	pnpm run watch

serve:
	pnpm run serve

update:
	pnpm update -i

test:
	pnpm run test

fmt:
	pnpm run prettier:fix

typecheck:
	pnpm run typecheck

lint: eslint prettier

eslint:
	pnpm run eslint:check

prettier:
	pnpm prettier:check

clean:
	rm -rf dist
	rm -rf node_modules
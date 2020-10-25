#!/bin/bash
mkdir -p dist/images
for f in src/public/images/*; do
	if [[ -f dist/images/${f##*/} ]]; then
		echo "Skipped ../dist/images/${f##*/}"
	else
		echo "Compressing src/public/images/${f##*/} -> dist/images/${f##*/}"
		pngquant --strip -skip-if-larger -o dist/images/"${f##*/}" "$f"
	fi
done

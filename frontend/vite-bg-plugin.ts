import { readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import sharp from "sharp";
import type { PluginOption } from "vite";

/**
 * Vite plugin that converts public/bg/*.png → dist/bg/*.webp during build,
 * then removes the original PNGs from dist/ to save space.
 */
export function bgWebPPlugin(): PluginOption {
	return {
		name: "bg-webp-conversion",
		enforce: "post",
		async closeBundle() {
			if (process.env.NODE_ENV === "development") return;

			const distBgDir = join(__dirname, "dist", "bg");
			const files = readdirSync(distBgDir).filter((f) => f.endsWith(".png"));

			let converted = 0;
			let deletedPngs = 0;
			let savedBytes = 0;

			for (const file of files) {
				const pngPath = join(distBgDir, file);
				const webpPath = join(distBgDir, file.replace(/\.png$/, ".webp"));

				const originalSize = statSync(pngPath).size;

				try {
					await sharp(readFileSync(pngPath)).webp({ quality: 75 }).toFile(webpPath);
					const newSize = statSync(webpPath).size;
					savedBytes += originalSize - newSize;

					// Remove the PNG after successful WebP conversion
					rmSync(pngPath);
					deletedPngs++;
					converted++;
				} catch {
					// Skip files that can't be converted, keep the PNG
				}
			}

			console.log(
				`[bg-webp] Converted ${converted}/${files.length} PNGs → WebP, removed ${deletedPngs} PNGs, saved ${(savedBytes / 1024 / 1024).toFixed(1)} MB`,
			);
		},
	};
}

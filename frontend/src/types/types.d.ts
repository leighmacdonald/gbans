declare module "*.png";
declare module "*.jpg";
declare module "*.jpeg";
declare module "*.gif";
declare module "*.svg";
declare module "*.webp";
declare module "*.webm";
declare module "*.mp3";

declare module "material-react-table" {
	interface MRT_TableOptions<TData extends import("material-react-table").MRT_RowData> {
		/**
		 * Enables the column pinning UI. Present in the v3.2.1 runtime but missing from the
		 * package's type definitions.
		 */
		enableColumnPinning?: boolean;
	}
}

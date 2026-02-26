import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { MuiMarkdown, type Options } from "mui-markdown";
import { type JSX, useMemo } from "react";
import { ImageBox } from "./ImageBox.tsx";
import RouterLink from "./RouterLink.tsx";

const renderLinks = (body_md: string, asset_url: string): string => {
	return body_md
		.replace("/^[\u200B\u200C\u200D\u200E\u200F\uFEFF]/", "")
		.replace(/(wiki:\/\/)/gi, "/wiki/")
		.replace(/(media:\/\/)/gi, asset_url !== "" ? asset_url : "/asset/" + "/");
};

interface MDImgProps {
	children: JSX.Element;
	src: string;
	alt: string;
	title: string;
}

const MDImg = ({ src, alt }: MDImgProps) => {
	return <ImageBox src={src} alt={alt} key={alt} maxWidth={"100%"} maxHeight={"100%"} />;
};

interface MDLnkProps {
	children: JSX.Element;
	href: string;
	title: string;
}

const MDLink = ({ children, href, title }: MDLnkProps) => {
	return (
		<Typography
			variant={"body1"}
			component={RouterLink}
			to={href}
			title={title}
			fontWeight={700}
			sx={{
				color: (theme) => theme.palette.text.primary,
				textDecoration: "none",
				"&:hover": { textDecoration: "underline" },
			}}
		>
			{children}
		</Typography>
	);
};

const mdRenderOpts: Options = {
	disableParsingRawHTML: true,
	overrides: {
		a: {
			component: MDLink,
		},
		img: {
			component: MDImg,
		},
		h1: {
			props: {
				variant: "h3",
			},
		},
		h2: {
			props: {
				variant: "h3",
			},
		},
		h3: {
			props: {
				variant: "h3",
			},
		},
	},
};

export const MarkDownRenderer = ({
	body_md,
	minHeight,
	assetURL,
}: {
	body_md: string;
	assetURL: string;
	minHeight?: number;
}) => {
	const links = useMemo(() => {
		return renderLinks(body_md, assetURL);
	}, [assetURL, body_md]);

	return (
		<Box padding={2} maxWidth={"100%"} minHeight={minHeight}>
			<MuiMarkdown options={mdRenderOpts}>{links}</MuiMarkdown>
		</Box>
	);
};

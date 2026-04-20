import Pagination from "@mui/material/Pagination";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";

import { useState } from "react";
import { renderDate, timestampToDateTime } from "../util/time.ts";
import { MarkDownRenderer } from "./MarkdownRenderer";
import { NewsHead } from "./NewsHead.tsx";
import { latest } from "../rpc/news/v1/news-NewsService_connectquery.ts";
import { useQuery } from "@connectrpc/connect-query";

interface NewsViewProps {
	itemsPerPage: number;
	assetURL: string;
}

export const NewsView = ({ itemsPerPage, assetURL }: NewsViewProps) => {
	const [page, setPage] = useState<number>(0);
	const { data, isLoading } = useQuery(latest, { limit: 1000 });

	return (
		<Stack spacing={2}>
			{!isLoading &&
				(data?.article ?? [])?.slice(page * itemsPerPage, page * itemsPerPage + itemsPerPage).map((article) => {
					if (!article.createdOn || !article.updatedOn) {
						return null;
					}
					return (
						<Paper elevation={1} key={`news_${article.newsId}`}>
							<NewsHead left={article.title} right={renderDate(timestampToDateTime(article.createdOn))} />
							<MarkDownRenderer body_md={article.bodyMd} assetURL={assetURL} />
						</Paper>
					);
				})}
			<Pagination
				count={data?.article ? Math.ceil(data.article.length / itemsPerPage) : 0}
				defaultValue={1}
				onChange={(_, newPage) => {
					setPage(newPage - 1);
				}}
			/>
		</Stack>
	);
};

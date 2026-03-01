import Pagination from "@mui/material/Pagination";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { apiGetNewsLatest } from "../api/news";
import { renderDate } from "../util/time.ts";
import { MarkDownRenderer } from "./MarkdownRenderer";
import { SplitHeading } from "./SplitHeading";

export interface NewsViewProps {
	itemsPerPage: number;
	assetURL: string;
}

export const NewsView = ({ itemsPerPage, assetURL }: NewsViewProps) => {
	const [page, setPage] = useState<number>(0);
	const { data: articles, isLoading } = useQuery({
		queryKey: ["articles"],
		queryFn: async () => {
			return await apiGetNewsLatest();
		},
	});

	return (
		<Stack spacing={2}>
			{!isLoading &&
				(articles ?? [])?.slice(page * itemsPerPage, page * itemsPerPage + itemsPerPage).map((article) => {
					if (!article.created_on || !article.updated_on) {
						return null;
					}
					return (
						<Paper elevation={1} key={`news_${article.news_id}`}>
							<SplitHeading left={article.title} right={renderDate(article.created_on)} />
							<MarkDownRenderer body_md={article.body_md} assetURL={assetURL} />
						</Paper>
					);
				})}
			<Pagination
				count={articles ? Math.ceil(articles.length / itemsPerPage) : 0}
				defaultValue={1}
				onChange={(_, newPage) => {
					setPage(newPage - 1);
				}}
			/>
		</Stack>
	);
};

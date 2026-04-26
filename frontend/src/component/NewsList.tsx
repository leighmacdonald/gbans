import { useQuery } from "@connectrpc/connect-query";
import FolderIcon from "@mui/icons-material/Folder";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import type { Article } from "../rpc/news/v1/news_pb.ts";
import { all } from "../rpc/news/v1/news-NewsService_connectquery.ts";
import { LoadingPlaceholder } from "./LoadingPlaceholder.tsx";

interface NewsListProps {
	setSelectedNewsEntry: (entry: Article) => void;
}

export const NewsList = ({ setSelectedNewsEntry }: NewsListProps) => {
	const theme = useTheme();

	const { data, isLoading } = useQuery(all);
	return (
		<Stack spacing={2} padding={2}>
			<List dense={true}>
				{isLoading ? (
					<LoadingPlaceholder />
				) : (
					(data?.articles ?? []).map((entry) => {
						return (
							<ListItem
								sx={[
									{
										"&:hover": {
											cursor: "pointer",
											backgroundColor: theme.palette.background.default,
										},
									},
								]}
								key={entry.newsId}
								onClick={() => {
									setSelectedNewsEntry(entry);
								}}
							>
								<ListItemIcon>
									<FolderIcon />
								</ListItemIcon>
								<ListItemText primary={entry.title} />
							</ListItem>
						);
					})
				)}
			</List>
		</Stack>
	);
};

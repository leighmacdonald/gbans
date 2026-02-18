import FolderIcon from "@mui/icons-material/Folder";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import { useQuery } from "@tanstack/react-query";
import { apiGetNewsAll } from "../api/news";
import type { NewsEntry } from "../schema/news.ts";
import { LoadingPlaceholder } from "./LoadingPlaceholder.tsx";

interface NewsListProps {
	setSelectedNewsEntry: (entry: NewsEntry) => void;
}

export const NewsList = ({ setSelectedNewsEntry }: NewsListProps) => {
	const theme = useTheme();

	const { data, isLoading } = useQuery({
		queryKey: ["newsList"],
		queryFn: async () => {
			return await apiGetNewsAll();
		},
	});
	return (
		<Stack spacing={2} padding={2}>
			<List dense={true}>
				{isLoading ? (
					<LoadingPlaceholder />
				) : (
					(data ?? []).map((entry) => {
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
								key={entry.news_id}
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

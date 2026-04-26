import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { Person2 } from "@mui/icons-material";
import AccessTimeIcon from "@mui/icons-material/AccessTime";
import TodayIcon from "@mui/icons-material/Today";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { recentMessages } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { avatarHashToURL } from "../../util/strings.ts";
import { renderTime, renderTimestamp } from "../../util/time.ts";
import { ContainerWithHeader } from "../ContainerWithHeader.tsx";
import { VCenteredElement } from "../Heading.tsx";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";
import RouterLink from "../RouterLink.tsx";
import { VCenterBox } from "../VCenterBox.tsx";
import { ForumRowLink } from "./ForumRowLink.tsx";

export const ForumRecentMessageActivity = () => {
	const { data, isLoading } = useQuery(recentMessages);

	return (
		<ContainerWithHeader title={"Latest Activity"} iconLeft={<TodayIcon />}>
			<Stack spacing={1}>
				{isLoading ? (
					<LoadingPlaceholder />
				) : (
					(data?.messages ?? []).map((m) => {
						return (
							<Stack
								direction={"row"}
								key={`message-${m.forumMessageId}`}
								spacing={1}
								sx={{
									overflow: "hidden",
									textOverflow: "ellipsis",
									whiteSpace: "nowrap",
									width: "100%",
								}}
							>
								<VCenteredElement
									icon={<Avatar alt={m.personaName} src={avatarHashToURL(m.avatarHash, "medium")} />}
								/>
								<Stack>
									<Box
										sx={{
											overflow: "hidden",
											textOverflow: "ellipsis",
											whiteSpace: "nowrap",
											width: "100%",
										}}
									>
										<ForumRowLink
											variant={"body1"}
											label={m.title ?? ""}
											to={`/forums/thread/${m.forumThreadId}#${m.forumMessageId}`}
										/>
									</Box>
									<Stack direction={"row"} spacing={1}>
										<AccessTimeIcon scale={0.5} />
										<VCenterBox>
											<Tooltip title={renderTimestamp(m.createdOn)}>
												<Typography variant={"body2"}>
													{renderTime(m.createdOn ? timestampDate(m.createdOn) : new Date())}
												</Typography>
											</Tooltip>
										</VCenterBox>
										<Person2 scale={0.5} />
										<VCenterBox>
											<Typography
												overflow={"hidden"}
												component={RouterLink}
												to={`/profile/${m.sourceId}`}
												variant={"body2"}
												sx={{
													color: (theme) => theme.palette.text.secondary,
													textDecoration: "none",
													"&:hover": {
														textDecoration: "underline",
													},
												}}
											>
												{m.personaName}
											</Typography>
										</VCenterBox>
									</Stack>
								</Stack>
							</Stack>
						);
					})
				)}
			</Stack>
		</ContainerWithHeader>
	);
};

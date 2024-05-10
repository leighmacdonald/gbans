import { Person2 } from '@mui/icons-material';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import TodayIcon from '@mui/icons-material/Today';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { apiForumRecentActivity } from '../api/forum.ts';
import { avatarHashToURL, renderDateTime, renderTime } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader.tsx';
import { ForumRowLink } from './ForumRowLink.tsx';
import { VCenteredElement } from './Heading.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import RouterLink from './RouterLink.tsx';
import { VCenterBox } from './VCenterBox.tsx';

export const ForumRecentMessageActivity = () => {
    const { data: messages, isLoading } = useQuery({
        queryKey: ['forumMessageActivity'],
        queryFn: async () => {
            return await apiForumRecentActivity();
        }
    });

    return (
        <ContainerWithHeader title={'Latest Activity'} iconLeft={<TodayIcon />}>
            <Stack spacing={1}>
                {isLoading ? (
                    <LoadingPlaceholder />
                ) : (
                    (messages ?? []).map((m) => {
                        return (
                            <Stack
                                direction={'row'}
                                key={`message-${m.forum_message_id}`}
                                spacing={1}
                                sx={{
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    width: '100%'
                                }}
                            >
                                <VCenteredElement
                                    icon={<Avatar alt={m.personaname} src={avatarHashToURL(m.avatarhash, 'medium')} />}
                                />
                                <Stack>
                                    <Box
                                        sx={{
                                            overflow: 'hidden',
                                            textOverflow: 'ellipsis',
                                            whiteSpace: 'nowrap',
                                            width: '100%'
                                        }}
                                    >
                                        <ForumRowLink
                                            variant={'body1'}
                                            label={m.title ?? ''}
                                            to={`/forums/thread/${m.forum_thread_id}#${m.forum_message_id}`}
                                        />
                                    </Box>
                                    <Stack direction={'row'} spacing={1}>
                                        <AccessTimeIcon scale={0.5} />
                                        <VCenterBox>
                                            <Tooltip title={renderDateTime(m.created_on)}>
                                                <Typography variant={'body2'}>
                                                    {renderTime(m.created_on ?? new Date())}
                                                </Typography>
                                            </Tooltip>
                                        </VCenterBox>
                                        <Person2 scale={0.5} />
                                        <VCenterBox>
                                            <Typography
                                                overflow={'hidden'}
                                                color={(theme) => {
                                                    return theme.palette.text.secondary;
                                                }}
                                                component={RouterLink}
                                                to={`/profile/${m.source_id}`}
                                                variant={'body2'}
                                                sx={{
                                                    textDecoration: 'none',
                                                    '&:hover': {
                                                        textDecoration: 'underline'
                                                    }
                                                }}
                                            >
                                                {m.personaname}
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

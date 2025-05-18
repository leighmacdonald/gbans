import { FormEvent, useCallback, useMemo, useState } from 'react';
import ChatBubbleIcon from '@mui/icons-material/ChatBubble';
import GroupIcon from '@mui/icons-material/Group';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import NavigateBeforeIcon from '@mui/icons-material/NavigateBefore';
import NavigateNextIcon from '@mui/icons-material/NavigateNext';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import Collapse from '@mui/material/Collapse';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { useAuth } from '../../hooks/useAuth.ts';
import { useQueueCtx } from '../../hooks/useQueueCtx.ts';
import { PermissionLevel } from '../../schema/people.ts';
import { emptyOrNullString } from '../../util/types.ts';
import { ContainerWithHeader } from '../ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../ContainerWithHeaderAndButtons.tsx';
import { VCenterBox } from '../VCenterBox.tsx';
import { QueueChatMessageContainer } from './QueueChatMessageContainer.tsx';
import { QueueChatName } from './QueueChatName.tsx';

export const QueueChat = () => {
    const { isReady, sendMessage, showChat, users, chatStatus, reason } = useQueueCtx();
    const { hasPermission, profile } = useAuth();
    const [showPeople, setShowPeople] = useState<boolean>(false);
    const [msg, setMsg] = useState<string>('');
    const [sending, setSending] = useState(false);
    const theme = useTheme();
    const matches = useMediaQuery(theme.breakpoints.down('md'));

    const onSubmit = useCallback(
        async (event: FormEvent<HTMLFormElement | HTMLDivElement | HTMLButtonElement>) => {
            event.preventDefault();

            if (msg.length > 0) {
                setSending(true);
                sendMessage(msg);
                setMsg('');
                setSending(false);
            }
        },
        [msg]
    );
    const isMod = hasPermission(PermissionLevel.Moderator);

    if (!isMod || profile.playerqueue_chat_status == 'noaccess') {
        return <></>;
    }

    const mq = useMemo(() => {
        if (matches || !showPeople) {
            return { xs: 12 };
        }
        return { md: showPeople ? 10 : 12, sm: 12, xs: 12 };
    }, [matches, showPeople]);

    const inputStates: { readonly: boolean; label: string; reason: string } = useMemo(() => {
        const readonly = profile.playerqueue_chat_status != 'readwrite';
        if (!isReady) {
            return { readonly: true, label: 'Connecting', reason: '' };
        }

        if (sending) {
            return { readonly: true, label: '...', reason: '' };
        }

        if (chatStatus == 'readonly') {
            return { readonly: true, label: 'Muted', reason: reason };
        }

        return { readonly: readonly, label: '', reason: '' };
    }, [isReady, sending, profile.playerqueue_chat_status, reason, chatStatus]);

    return (
        <Collapse in={showChat && chatStatus != 'noaccess'}>
            <Grid container spacing={1} sx={{ marginBottom: 3 }}>
                <Grid size={mq}>
                    <ContainerWithHeaderAndButtons
                        title={'Queue Lobby Chat'}
                        iconLeft={<ChatBubbleIcon />}
                        buttons={[
                            <Button
                                endIcon={showPeople ? <NavigateNextIcon /> : <NavigateBeforeIcon />}
                                size={'small'}
                                variant={'contained'}
                                color={!showPeople ? 'success' : 'error'}
                                style={{ height: 18 }}
                                key={'show-players-button'}
                                onClick={() => {
                                    setShowPeople((prevState) => !prevState);
                                }}
                            >
                                {!showPeople ? 'Show Players' : 'Hide Players'}
                            </Button>
                        ]}
                    >
                        <Stack maxHeight={200} minHeight={200}>
                            <QueueChatMessageContainer showControls={isMod} />
                            <form onSubmit={onSubmit}>
                                <Stack spacing={1} direction={'row'}>
                                    {chatStatus == 'readonly' ? (
                                        <div style={{ display: 'inline-block', width: '100%' }}>
                                            <VCenterBox>
                                                <Typography
                                                    variant={'button'}
                                                    fontSize={10}
                                                    color={'textSecondary'}
                                                    textAlign={'center'}
                                                >
                                                    {`You are currently muted`}
                                                </Typography>
                                            </VCenterBox>
                                        </div>
                                    ) : (
                                        <TextField
                                            disabled={inputStates.readonly}
                                            onSubmit={onSubmit}
                                            fullWidth={true}
                                            size={'small'}
                                            placeholder={inputStates.reason != '' ? inputStates.reason : undefined}
                                            name={'msg'}
                                            value={msg}
                                            onChange={(event) => {
                                                setMsg(event.target.value);
                                            }}
                                        />
                                    )}

                                    <IconButton
                                        disabled={inputStates.readonly}
                                        size={'small'}
                                        color={msg.length > 0 ? 'success' : 'default'}
                                        onClick={onSubmit}
                                    >
                                        {inputStates.readonly ? <HourglassBottomIcon /> : <SendIcon />}
                                    </IconButton>
                                </Stack>
                            </form>
                        </Stack>
                    </ContainerWithHeaderAndButtons>
                </Grid>
                {showPeople && (
                    <Grid size={{ sm: 12, md: 2 }}>
                        <ContainerWithHeader title={`Online`} iconLeft={<GroupIcon />}>
                            <Grid
                                container
                                maxHeight={200}
                                minHeight={200}
                                overflow={'auto'}
                                direction={'column'}
                                padding={1}
                            >
                                {users.map((u) => {
                                    return (
                                        <Grid size={{ xs: 4, md: 12 }} key={`memberlist-${u.steam_id}`}>
                                            <QueueChatName
                                                personaname={emptyOrNullString(u.name) ? u.steam_id : u.name}
                                                steam_id={u.steam_id}
                                                avatarhash={u.hash}
                                            />
                                        </Grid>
                                    );
                                })}
                                {}
                            </Grid>
                        </ContainerWithHeader>
                    </Grid>
                )}
            </Grid>
        </Collapse>
    );
};

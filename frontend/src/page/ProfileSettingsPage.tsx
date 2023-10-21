import AddLinkIcon from '@mui/icons-material/AddLink';
import ChatBubbleIcon from '@mui/icons-material/ChatBubble';
import LinkIcon from '@mui/icons-material/Link';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import React, { JSX } from 'react';
import { discordLoginURL } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const ProfileSettingsPage = (): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();
    const loginUrl = discordLoginURL();
    return (
        <ContainerWithHeader
            title={'Discord Settings'}
            iconLeft={<ChatBubbleIcon />}
        >
            <Stack padding={2} paddingTop={0} spacing={2}>
                <Tooltip title={`id: ${currentUser.steam_id}`}>
                    <Button
                        component={Link}
                        href={loginUrl}
                        variant={'contained'}
                        disabled={currentUser.discord_id != ''}
                        startIcon={
                            currentUser.discord_id == '' ? (
                                <AddLinkIcon />
                            ) : (
                                <LinkIcon />
                            )
                        }
                    >
                        {currentUser.discord_id != ''
                            ? 'Already Linked!'
                            : 'Link Discord'}
                    </Button>
                </Tooltip>
                <Typography variant={'body1'}>
                    By linking your discord account, you will unlock certain
                    functionality available on the discord platform. This
                    currently includes functionality related to reporting
                    primarily, but will be extended to include other
                    functionality in the future.
                </Typography>
            </Stack>
        </ContainerWithHeader>
    );
};

import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import PersonIcon from '@mui/icons-material/Person';
import SendIcon from '@mui/icons-material/Send';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { GameStartPayload } from '../../schema/playerqueue.ts';
import { logErr } from '../../util/errors.ts';
import { avatarHashToURL } from '../../util/text.tsx';
import { ButtonLink } from '../ButtonLink.tsx';
import { Heading } from '../Heading';

export const QueueJoinModal = NiceModal.create(({ gameStart }: { gameStart: GameStartPayload }) => {
    const modal = useModal();

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
            <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                Queue Ready! ({gameStart.users.length} Players)
            </DialogTitle>

            <DialogContent>
                <Typography variant={'subtitle1'} textAlign={'center'}>
                    You are on your way to
                </Typography>
                <Typography variant={'h4'} textAlign={'center'}>
                    {gameStart.server.name}
                </Typography>
                <Grid container spacing={2} padding={2}>
                    {gameStart.users.map((p) => {
                        return (
                            <Grid
                                size={{ xs: 4 }}
                                display="flex"
                                justifyContent="center"
                                alignItems="center"
                                key={`queue-player-${p.steam_id}`}
                            >
                                <Avatar
                                    sx={{ height: 64, width: 64 }}
                                    alt={p.name}
                                    src={avatarHashToURL(p.hash, 'full')}
                                />
                            </Grid>
                        );
                    })}
                </Grid>
                <Typography variant={'h5'} fontFamily={'monospace'} textAlign={'center'}>
                    {gameStart.server.connect_command}
                </Typography>
            </DialogContent>
            <DialogActions>
                <Grid container>
                    <Grid size={{ xs: 12 }}>
                        <ButtonGroup fullWidth={false}>
                            <Button
                                color={'success'}
                                variant={'contained'}
                                startIcon={<ContentCopyIcon />}
                                onClick={async () => {
                                    try {
                                        await navigator.clipboard.writeText(gameStart.server.connect_command);
                                    } catch (e) {
                                        logErr(e);
                                    }
                                }}
                            >
                                Copy Command
                            </Button>
                            <ButtonLink
                                to={gameStart.server.connect_url}
                                key={'submit-button'}
                                type="button"
                                variant={'contained'}
                                color={'success'}
                                startIcon={<SendIcon />}
                            >
                                Connect
                            </ButtonLink>
                        </ButtonGroup>
                    </Grid>
                </Grid>
            </DialogActions>
        </Dialog>
    );
});

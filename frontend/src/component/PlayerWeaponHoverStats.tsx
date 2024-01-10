import React from 'react';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { Popover } from '@mui/material';
import Box from '@mui/material/Box';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { MatchPlayerWeapon } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import { WeaponStatRow } from './WeaponStatRow';

interface PlayerWeaponHoverStatsProps {
    stats: MatchPlayerWeapon[];
}

export const PlayerWeaponHoverStats = ({
    stats
}: PlayerWeaponHoverStatsProps) => {
    const [anchorEl, setAnchorEl] = React.useState<HTMLElement | null>(null);

    const handlePopoverOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handlePopoverClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    return (
        <Box>
            <Box
                display="flex"
                justifyContent="right"
                alignItems="center"
                onMouseEnter={handlePopoverOpen}
                onMouseLeave={handlePopoverClose}
            >
                <InfoOutlinedIcon />
            </Box>
            <Popover
                id="mouse-over-popover"
                sx={{
                    pointerEvents: 'none'
                }}
                open={open}
                anchorEl={anchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left'
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
                onClose={handlePopoverClose}
                disableRestoreFocus
            >
                <ContainerWithHeader title={'Weapon Stats'}>
                    <TableContainer>
                        <Table padding={'checkbox'} size={'small'}>
                            <TableHead>
                                <TableRow>
                                    <TableCell variant="head" width={'400px'}>
                                        <Typography variant={'button'}>
                                            Weapon
                                        </Typography>
                                    </TableCell>
                                    <TableCell variant="head">
                                        <Typography variant={'button'}>
                                            Kills
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Damage
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Shots
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Hits
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            Acc%
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            BS
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            HS
                                        </Typography>
                                    </TableCell>
                                    <TableCell>
                                        <Typography variant={'button'}>
                                            AS
                                        </Typography>
                                    </TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {stats.map((ws, index) => {
                                    return (
                                        <WeaponStatRow
                                            weaponStat={ws}
                                            key={`ws-${ws.damage}-${ws.weapon_id}-${index}`}
                                        />
                                    );
                                })}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </ContainerWithHeader>
            </Popover>
        </Box>
    );
};

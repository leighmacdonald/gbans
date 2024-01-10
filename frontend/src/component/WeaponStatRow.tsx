import React from 'react';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { MatchPlayerWeapon } from '../api';

interface WeaponStatRowProps {
    weaponStat: MatchPlayerWeapon;
}

const WeaponCell = ({
    value,
    width
}: {
    value: string | number;
    width?: number | string;
}) => {
    return (
        <TableCell width={width ?? 'auto'}>
            <Typography
                padding={0.5}
                variant={'body2'}
                sx={{ fontFamily: 'Monospace' }}
            >
                {value}
            </Typography>
        </TableCell>
    );
};

export const WeaponStatRow = ({ weaponStat }: WeaponStatRowProps) => {
    return (
        <TableRow>
            <WeaponCell value={weaponStat.name} width={'400px'} />
            <WeaponCell value={weaponStat.kills} />
            <WeaponCell value={weaponStat.damage} />
            <WeaponCell value={weaponStat.shots} />
            <WeaponCell value={weaponStat.hits} />
            <WeaponCell
                value={`${
                    !isNaN((weaponStat.hits / weaponStat.shots) * 100)
                        ? ((weaponStat.hits / weaponStat.shots) * 100).toFixed(
                              2
                          )
                        : 0
                }%`}
            />
            <WeaponCell value={weaponStat.backstabs} />
            <WeaponCell value={weaponStat.headshots} />
            <WeaponCell value={weaponStat.airshots} />
        </TableRow>
    );
};

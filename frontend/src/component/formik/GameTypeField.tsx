import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export enum GameType {
    sixes = 'sixes',
    highlander = 'highlander',
    ultiduo = 'ultiduo'
}

export const GameTypes = [
    GameType.sixes,
    GameType.highlander,
    GameType.ultiduo
];

export const gameTypeValidator = yup
    .string()
    .test('checkGameType', 'Invalid game type selection', async (gameType) => {
        return GameTypes.includes(gameType as GameType);
    })
    .label('Select a game type to play')
    .required('game type is required');

export const GameTypeField = ({
    formik
}: {
    formik: FormikState<{
        gameType: GameType;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="gameType-label">Game Type</InputLabel>
            <Select<GameType>
                fullWidth
                label={'Game Type'}
                labelId="gameType-label"
                id="gameType"
                name={'gameType'}
                value={formik.values.gameType}
                onChange={formik.handleChange}
                error={
                    formik.touched.gameType && Boolean(formik.errors.gameType)
                }
                defaultValue={GameType.sixes}
            >
                {GameTypes.map((v) => (
                    <MenuItem key={`gameType-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.gameType && formik.errors.gameType}
            </FormHelperText>
        </FormControl>
    );
};

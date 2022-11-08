import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import React from 'react';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import * as yup from 'yup';

export enum GameConfig {
    etf2l = 'etf2l',
    rgl = 'rgl',
    ugc = 'ugc',
    ozfortress = 'ozfortress'
}

export const GameConfigs = [
    GameConfig.etf2l,
    GameConfig.rgl,
    GameConfig.ugc,
    GameConfig.ozfortress
];

export const gameConfigValidator = yup
    .string()
    .test(
        'checkGameConfig',
        'Invalid game type selection',
        async (gameConfig) => GameConfigs.includes(gameConfig as GameConfig)
    )
    .label('Select a config to use')
    .required('game config is required');

export const GameConfigField = ({
    formik
}: {
    formik: FormikState<{
        gameConfig: GameConfig;
    }> &
        FormikHandlers;
}) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="actionType-label">Game Config</InputLabel>
            <Select<GameConfig>
                fullWidth
                label={'Game Config'}
                labelId="gameConfig-label"
                id="gameConfig"
                name={'gameConfig'}
                value={formik.values.gameConfig}
                onChange={formik.handleChange}
                error={
                    formik.touched.gameConfig &&
                    Boolean(formik.errors.gameConfig)
                }
                defaultValue={GameConfig.rgl}
            >
                {GameConfigs.map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {formik.touched.gameConfig && formik.errors.gameConfig}
            </FormHelperText>
        </FormControl>
    );
};

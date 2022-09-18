import { SelectChangeEvent } from '@mui/material/Select';
import React from 'react';
import { useState } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import { Checkbox, OutlinedInput, Select } from '@mui/material';
import { qpGameType, qpKnownGameTypes } from '../api';
import MenuItem from '@mui/material/MenuItem';
import ListItemText from '@mui/material/ListItemText';

const ITEM_HEIGHT = 48;
const ITEM_PADDING_TOP = 8;
const MenuProps = {
    PaperProps: {
        style: {
            maxHeight: ITEM_HEIGHT * 4.5 + ITEM_PADDING_TOP,
            width: 250
        }
    }
};

export interface GameModeSelectProps {
    onChange: (newModes: qpGameType[]) => void;
}

export const GameModeSelect = ({ onChange }: GameModeSelectProps) => {
    const [gameTypes, setGameTypes] = useState<string[]>([]);

    const handleChange = (event: SelectChangeEvent<string[]>) => {
        const {
            target: { value }
        } = event;
        const newValues = typeof value === 'string' ? value.split(',') : value;
        setGameTypes(newValues);
        onChange(qpKnownGameTypes.filter((gt) => gameTypes.includes(gt.name)));
    };

    return (
        <div>
            <FormControl sx={{ m: 1, width: 300 }}>
                <InputLabel id="game-types-label">
                    Allowed Game Types
                </InputLabel>
                <Select<string[]>
                    labelId="game-types-label"
                    id="game-types-checkbox"
                    multiple
                    value={gameTypes}
                    onChange={handleChange}
                    input={<OutlinedInput label="Allowed Game Types" />}
                    renderValue={(selected) => selected.join(', ')}
                    MenuProps={MenuProps}
                >
                    {qpKnownGameTypes.map((gameType) => (
                        <MenuItem key={gameType.name} value={gameType.name}>
                            <Checkbox
                                checked={
                                    gameTypes.filter((t) => t == gameType.name)
                                        .length > 0
                                }
                            />
                            <ListItemText primary={gameType.name} />
                        </MenuItem>
                    ))}
                </Select>
            </FormControl>
        </div>
    );
};

import React from 'react';
import { App } from './App';
import { createRoot } from 'react-dom/client';
import { PaletteMode } from '@mui/material';

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(
        <App
            initialTheme={
                (localStorage.getItem('theme') as PaletteMode) || 'light'
            }
        />
    );
}

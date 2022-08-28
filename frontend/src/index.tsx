import React from 'react';
import { App } from './App';
import { createRoot } from 'react-dom/client';
import { tokenKey } from './api';
import { PaletteMode } from '@mui/material';

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(
        <App
            initialToken={localStorage.getItem(tokenKey) || ''}
            initialTheme={
                (localStorage.getItem('theme') as PaletteMode) || 'light'
            }
        />
    );
}

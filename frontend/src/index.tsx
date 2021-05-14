import React from 'react';
import ReactDOM from 'react-dom';
import { App } from './App';
import { CssBaseline } from '@material-ui/core';
import ThemeProvider from './themes/provider';

ReactDOM.render(
    <ThemeProvider>
        <CssBaseline />
        <React.StrictMode>
            <App />
        </React.StrictMode>
    </ThemeProvider>,
    document.getElementById('root')
);

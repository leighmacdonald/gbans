const path = require('path');

const CWD = process.cwd();

module.exports = {
    "includePaths": [
        path.resolve(CWD, 'node_modules'),
        path.resolve(CWD, 'src'),
        // This allows us to use relative import paths for foundation
        // Add this path to the projects IDEs resource root to get auto-completion to work
        path.resolve(CWD, 'node_modules/foundation-sites/scss')
    ]
};
import * as path from 'path';
import * as webpack from 'webpack';
import * as copyWebpackPlugin from 'copy-webpack-plugin';

const config: webpack.Configuration = {
    entry: './src/index.tsx',
    devtool: "source-map",
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
            {
                test: /\.(scss|css)$/,
                use: ['style-loader', 'css-loader', 'sass-loader'],
            },
            {
                test: /\.(jpg|png|woff|otf|ttf|svg|eot)$/,
                type: 'asset/resource',
                generator: {
                    filename: 'static/[hash][ext][query]'
                }
            },
        ],
    },
    resolve: {
        extensions: [ '.tsx', '.ts', '.js' ],
    },
    plugins: [
        new copyWebpackPlugin({
            patterns: [
                { from: 'src/icons' }
            ]
        })
    ],
    output: {
        filename: 'bundle.js',
        path: path.resolve(__dirname, 'dist'),
        sourceMapFilename: "[name].ts.map",
        assetModuleFilename: 'images/[hash][ext][query]'
    },
};

// noinspection JSUnusedGlobalSymbols
export default config;
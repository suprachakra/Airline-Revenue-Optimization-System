const path = require('path');
const SriPlugin = require('webpack-subresource-integrity');
const cspConfig = require('./config/csp-headers.config');
const securityHeaders = require('./config/security-headers.js');

module.exports = {
  entry: './src/index.js',
  output: {
    path: path.resolve(__dirname, 'build'),
    filename: 'bundle.js',
    crossOriginLoading: 'anonymous'
  },
  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        exclude: /node_modules/,
        use: ['babel-loader']
      },
      {
        test: /\.(png|jpe?g|gif)$/i,
        use: [{
          loader: 'responsive-loader',
          options: {
            adapter: require('responsive-loader/sharp'),
            sizes: [360, 768, 1280],
            format: 'webp'
          }
        }]
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      }
    ]
  },
  plugins: [
    new SriPlugin({
      hashFuncNames: ['sha384'],
      enabled: process.env.NODE_ENV === 'production'
    })
  ],
  resolve: {
    extensions: ['.js', '.jsx']
  },
  devServer: {
    static: './build',
    hot: true
  },
  performance: {
    hints: false
  }
};

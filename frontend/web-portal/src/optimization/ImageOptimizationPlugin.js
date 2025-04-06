// Webpack plugin configuration for optimizing images.
module.exports = {
  module: {
    rules: [
      {
        test: /\.(png|jpe?g)$/i,
        use: [
          {
            loader: 'responsive-loader',
            options: {
              adapter: require('responsive-loader/sharp'),
              sizes: [360, 768, 1280],
              format: 'webp',
            },
          },
        ],
      },
    ],
  },
};

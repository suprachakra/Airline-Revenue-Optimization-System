module.exports = {
  directives: {
    defaultSrc: ["'self'"],
    scriptSrc: ["'self'", "'sha256-YourHashHere'", "https://www.google-analytics.com"],
    styleSrc: ["'self'", "'unsafe-inline'"],
    imgSrc: ["'self'", "data:", "https://*.iaros.ai"],
    connectSrc: ["'self'", "https://api.iaros.ai"],
    frameAncestors: ["'none'"],
    formAction: ["'self'"]
  },
  reportOnly: false,
  featurePolicy: {
    geolocation: "'none'"
  }
};

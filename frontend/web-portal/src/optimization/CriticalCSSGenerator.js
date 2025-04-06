const fs = require('fs');
const penthouse = require('penthouse');

penthouse({
  url: 'http://localhost:3000',
  css: 'build/main.css'
}, (err, criticalCss) => {
  if (err) {
    console.error('Critical CSS generation failed:', err);
  } else {
    fs.writeFileSync('build/critical.css', criticalCss);
    console.log('Critical CSS generated successfully.');
  }
});

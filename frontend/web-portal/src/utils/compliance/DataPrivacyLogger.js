export const sanitizeLog = (data) => {
  return JSON.stringify(data)
    .replace(/"email":"(.*?)"/g, '"email":"[REDACTED]"')
    .replace(/"userId":"(.*?)"/g, '"userId":"[REDACTED]"');
};

export const logDataPrivacyEvent = (event) => {
  const sanitized = sanitizeLog(event);
  console.log('Data Privacy Log:', sanitized);
  // In production, send the sanitized log to a centralized logging system.
};

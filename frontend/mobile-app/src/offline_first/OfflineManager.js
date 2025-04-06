import AsyncStorage from '@react-native-async-storage/async-storage';

export const cacheData = async (key, data, ttl = 259200) => {
  // ttl default is 72 hours in seconds
  const record = { data, timestamp: Date.now(), ttl };
  await AsyncStorage.setItem(`cache:${key}`, JSON.stringify(record));
};

export const getCachedData = async (key) => {
  const recordStr = await AsyncStorage.getItem(`cache:${key}`);
  if (!recordStr) return null;
  const record = JSON.parse(recordStr);
  const isStale = (Date.now() - record.timestamp) / 1000 > record.ttl;
  return isStale ? null : record.data;
};

export const clearCache = async (key) => {
  await AsyncStorage.removeItem(`cache:${key}`);
};

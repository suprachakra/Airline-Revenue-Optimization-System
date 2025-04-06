import React from 'react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

export default function ForecastChart({ data }) {
  if (!data || data.length === 0) {
    return <div>No forecast data available.</div>;
  }
  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={data}>
        <XAxis dataKey="date" />
        <YAxis />
        <Tooltip />
        <Line type="monotone" dataKey="forecast" stroke="#003366" />
        <Line type="monotone" dataKey="actual" stroke="#ff6600" />
      </LineChart>
    </ResponsiveContainer>
  );
}

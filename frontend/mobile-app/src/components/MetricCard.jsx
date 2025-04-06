import React from 'react';
import PropTypes from 'prop-types';

export default function MetricCard({ metric, value, unit }) {
  return (
    <div className="metric-card">
      <h3>{metric}</h3>
      <p>{value} {unit}</p>
    </div>
  );
}

MetricCard.propTypes = {
  metric: PropTypes.string.isRequired,
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
  unit: PropTypes.string
};

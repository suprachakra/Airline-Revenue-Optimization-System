import React from 'react';
import { Table, TableHead, TableBody, TableRow, TableCell } from '@mui/material';

export default function ReportsPage({ reportData }) {
  if (!reportData) return <div>Loading report data...</div>;

  return (
    <div className="reports-page">
      <h2>Revenue and Performance Reports</h2>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Metric</TableCell>
            <TableCell>Current</TableCell>
            <TableCell>Target</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {reportData.map((metric, idx) => (
            <TableRow key={idx}>
              <TableCell>{metric.name}</TableCell>
              <TableCell>{metric.current}</TableCell>
              <TableCell>{metric.target}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <p>
        For troubleshooting, refer to the [Technical Runbook](../runbooks/reports_troubleshooting_v3.md).
      </p>
    </div>
  );
}

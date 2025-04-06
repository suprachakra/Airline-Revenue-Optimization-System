import React, { Component } from 'react';
import Alert from '@mui/material/Alert';
import PersonalizedFallback from './PersonalizedFallback';

export class GlobalErrorBoundary extends Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }
  
  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }
  
  componentDidCatch(error, info) {
    console.error("GlobalErrorBoundary caught an error:", error, info);
  }
  
  render() {
    if (this.state.hasError) {
      return <PersonalizedFallback error={this.state.error} />;
    }
    return this.props.children;
  }
}
export default GlobalErrorBoundary;

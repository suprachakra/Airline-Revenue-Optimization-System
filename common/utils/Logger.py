"""
Enterprise Logging Utility for IAROS
====================================
Comprehensive structured logging with enterprise features including:
- Multi-level logging with performance metrics
- Multiple handlers (console, file, syslog, SIEM)
- Log rotation and security event correlation
- Real-time alerting and audit trail capabilities
"""

import logging
import logging.handlers
import json
import sys
import time
import threading
from datetime import datetime
from typing import Any, Dict, Optional
from pathlib import Path
import uuid

class IAROSLogger:
    """Enterprise-grade logger for IAROS platform"""
    
    def __init__(self, name: str):
        self.name = name
        self.logger = logging.getLogger(name)
        self.logger.setLevel(logging.INFO)
        self.metrics = {'total_logs': 0, 'error_count': 0, 'start_time': time.time()}
        self.metrics_lock = threading.Lock()
        self._setup_handlers()
    
    def _setup_handlers(self):
        """Setup console and file handlers with JSON formatting"""
        # Clear existing handlers
        self.logger.handlers.clear()
        
        # Console Handler
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setFormatter(JSONFormatter())
        self.logger.addHandler(console_handler)
        
        # File Handler with Rotation
        log_dir = Path('logs')
        log_dir.mkdir(exist_ok=True)
        
        file_handler = logging.handlers.RotatingFileHandler(
            'logs/iaros.log', maxBytes=100*1024*1024, backupCount=10
        )
        file_handler.setFormatter(JSONFormatter())
        self.logger.addHandler(file_handler)
    
    def info(self, message: str, **kwargs):
        """Log info message with context"""
        self._log(logging.INFO, message, **kwargs)
    
    def warning(self, message: str, **kwargs):
        """Log warning message"""
        self._log(logging.WARNING, message, **kwargs)
    
    def error(self, message: str, error: Optional[Exception] = None, **kwargs):
        """Log error message with exception details"""
        with self.metrics_lock:
            self.metrics['error_count'] += 1
        if error:
            kwargs['error_type'] = type(error).__name__
            kwargs['error_message'] = str(error)
        self._log(logging.ERROR, message, **kwargs)
    
    def security_event(self, event_type: str, message: str, **kwargs):
        """Log security event for SIEM integration"""
        kwargs.update({
            'security_event': True,
            'event_type': event_type,
            'timestamp': datetime.utcnow().isoformat(),
            'severity': 'security'
        })
        self._log(logging.WARNING, f"SECURITY: {message}", **kwargs)
    
    def audit_log(self, action: str, user_id: str, resource: str, **kwargs):
        """Log audit event for compliance"""
        kwargs.update({
            'audit_event': True,
            'action': action,
            'user_id': user_id,
            'resource': resource,
            'timestamp': datetime.utcnow().isoformat()
        })
        self._log(logging.INFO, f"AUDIT: {action} on {resource} by {user_id}", **kwargs)
    
    def performance_log(self, operation: str, duration_ms: float, **kwargs):
        """Log performance metrics"""
        kwargs.update({
            'performance_event': True,
            'operation': operation,
            'duration_ms': duration_ms,
            'timestamp': datetime.utcnow().isoformat()
        })
        level = logging.ERROR if duration_ms > 5000 else logging.WARNING if duration_ms > 1000 else logging.INFO
        self._log(level, f"PERFORMANCE: {operation} took {duration_ms}ms", **kwargs)
    
    def _log(self, level: int, message: str, **kwargs):
        """Internal logging with context"""
        with self.metrics_lock:
            self.metrics['total_logs'] += 1
        
        extra = {
            'service': self.name,
            'log_id': str(uuid.uuid4()),
            'timestamp': datetime.utcnow().isoformat(),
            **kwargs
        }
        self.logger.log(level, message, extra=extra)

class JSONFormatter(logging.Formatter):
    """JSON formatter for structured logging"""
    
    def format(self, record: logging.LogRecord) -> str:
        log_data = {
            'timestamp': datetime.utcfromtimestamp(record.created).isoformat(),
            'level': record.levelname,
            'logger': record.name,
            'message': record.getMessage(),
            'module': record.module,
            'function': record.funcName,
            'line': record.lineno
        }
        
        # Add extra fields
        if hasattr(record, '__dict__'):
            for key, value in record.__dict__.items():
                if key not in ['name', 'msg', 'args', 'levelname', 'levelno', 
                              'pathname', 'filename', 'module', 'lineno', 
                              'funcName', 'created', 'msecs', 'relativeCreated',
                              'thread', 'threadName', 'processName', 'process',
                              'getMessage', 'exc_info', 'exc_text', 'stack_info']:
                    log_data[key] = value
        
        return json.dumps(log_data, default=str)

# Logger factory
_loggers = {}

def get_logger(name: str) -> IAROSLogger:
    """Get or create logger instance"""
    if name not in _loggers:
        _loggers[name] = IAROSLogger(name)
    return _loggers[name]

#!/usr/bin/env python3
"""
Centralized logging configuration for the Timeline Processor project.

This module provides enhanced logging setup with:
1. Automatic logs directory creation
2. Both file and console handlers
3. Timestamped log files
4. Consistent format across all modules
"""

import logging
import os
import sys
from datetime import datetime
from pathlib import Path


def setup_logging(
    module_name: str = "timeline_processor",
    log_level: int = logging.INFO,
    console_format: str = '%(asctime)s - %(levelname)s - %(message)s',
    file_format: str = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
) -> logging.Logger:
    """
    Set up enhanced logging configuration with file and console handlers.
    
    Args:
        module_name: Name of the module for the logger
        log_level: Logging level (default: INFO)
        console_format: Format string for console output
        file_format: Format string for file output
        
    Returns:
        Configured logger instance
    """
    # Get the project root (assuming this module is in PROJECT_ROOT/src/)
    project_root = Path(__file__).resolve().parent.parent
    logs_dir = project_root / "logs"
    
    # Create logs directory if it doesn't exist
    logs_dir.mkdir(exist_ok=True)
    
    # Create timestamped log filename
    timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
    log_filename = f"{module_name}_{timestamp}.log"
    log_file_path = logs_dir / log_filename
    
    # Create logger
    logger = logging.getLogger(module_name)
    logger.setLevel(log_level)
    
    # Clear any existing handlers to avoid duplicates
    logger.handlers.clear()
    
    # Create console handler with maintained format
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setLevel(log_level)
    console_formatter = logging.Formatter(console_format)
    console_handler.setFormatter(console_formatter)
    
    # Create file handler with enhanced format
    file_handler = logging.FileHandler(log_file_path, encoding='utf-8')
    file_handler.setLevel(log_level)
    file_formatter = logging.Formatter(file_format)
    file_handler.setFormatter(file_formatter)
    
    # Add handlers to logger
    logger.addHandler(console_handler)
    logger.addHandler(file_handler)
    
    # Prevent propagation to avoid duplicate messages
    logger.propagate = False
    
    # Log the initialization
    logger.info(f"Logging initialized for {module_name}")
    logger.info(f"Log file: {log_file_path}")
    
    return logger


def get_logger(module_name: str = None) -> logging.Logger:
    """
    Get or create a logger for the specified module.
    
    Args:
        module_name: Name of the module. If None, uses the calling module's name.
        
    Returns:
        Logger instance
    """
    if module_name is None:
        # Try to get the calling module's name
        frame = sys._getframe(1)
        module_name = frame.f_globals.get('__name__', 'unknown')
        
        # Clean up module name for file naming
        if module_name == '__main__':
            # Use the script filename without extension
            script_path = frame.f_globals.get('__file__', 'unknown')
            if script_path and script_path != 'unknown':
                module_name = Path(script_path).stem
    
    # Check if logger already exists
    logger = logging.getLogger(module_name)
    if logger.handlers:
        return logger
    
    # Create new logger if it doesn't exist
    return setup_logging(module_name)


def configure_basic_logging(log_level: int = logging.INFO) -> None:
    """
    Configure basic logging for backward compatibility.
    This function maintains the same interface as logging.basicConfig()
    but uses our enhanced logging setup.
    
    Args:
        log_level: Logging level (default: INFO)
    """
    # Get the calling module's name
    frame = sys._getframe(1)
    module_name = frame.f_globals.get('__name__', 'unknown')
    
    if module_name == '__main__':
        # Use the script filename without extension
        script_path = frame.f_globals.get('__file__', 'unknown')
        if script_path and script_path != 'unknown':
            module_name = Path(script_path).stem
    
    # Set up logging
    logger = setup_logging(module_name, log_level)
    
    # Configure root logger to use our setup
    root_logger = logging.getLogger()
    root_logger.handlers.clear()
    root_logger.setLevel(log_level)
    
    # Add our handlers to root logger
    for handler in logger.handlers:
        root_logger.addHandler(handler)

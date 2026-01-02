#!/bin/bash

# TraceIQ Timeline Processor Launcher
cd "$(dirname "$0")"

# Check if virtual environment exists
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install dependencies if not already installed
if ! python -c "import flask" &> /dev/null; then
    echo "Installing dependencies..."
    pip install -r requirements.txt
fi

# Create database directory if needed
mkdir -p data/processed
mkdir -p data/logs

echo "Starting TraceIQ Timeline Processor..."
echo "Open your browser to: http://localhost:5000"
echo "Logs available at: data/logs/app.log"

python app.py "$@"

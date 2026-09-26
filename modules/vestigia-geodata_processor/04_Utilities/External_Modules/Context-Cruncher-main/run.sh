#!/bin/bash
# Context Cruncher - Launch Script
# Wrapper script to easily launch the Context Cruncher application

set -e  # Exit on error

echo "🎙️  Context Cruncher - Launch Script"
echo "===================================="
echo ""

# Check if virtual environment exists
if [ ! -d ".venv" ]; then
    echo "❌ Virtual environment not found!"
    echo "Creating virtual environment with uv..."
    uv venv
    echo "✅ Virtual environment created"
    echo ""
fi

# Activate virtual environment
echo "📦 Activating virtual environment..."
source .venv/bin/activate

# Check if dependencies are installed
if ! python -c "import gradio" 2>/dev/null; then
    echo "📥 Installing dependencies..."
    uv pip install -r requirements.txt
    echo "✅ Dependencies installed"
    echo ""
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "⚠️  Warning: .env file not found!"
    echo "Please create a .env file with your Gemini API key."
    echo "You can copy .env.example and add your key:"
    echo ""
    echo "  cp .env.example .env"
    echo "  # Then edit .env and add your GEMINI_API key"
    echo ""
    read -p "Do you want to continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Launch the application
echo "🚀 Launching Context Cruncher..."
echo "The app will open in your browser at http://localhost:7860"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

python app.py

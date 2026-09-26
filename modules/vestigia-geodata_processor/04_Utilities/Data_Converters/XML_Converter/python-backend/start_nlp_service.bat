@echo off
echo Starting Salem Forensic NLP Service...
echo.

REM Install dependencies if needed
if not exist "venv\" (
    echo Creating virtual environment...
    python -m venv venv
)

REM Activate virtual environment
call venv\Scripts\activate.bat

REM Install/upgrade requirements
echo Installing dependencies...
pip install -r requirements.txt

REM Download spaCy model if not present
python -m spacy download en_core_web_sm

REM Start the service
echo.
echo Starting service on http://localhost:8000
echo Press Ctrl+C to stop
echo.
python nlp_service.py

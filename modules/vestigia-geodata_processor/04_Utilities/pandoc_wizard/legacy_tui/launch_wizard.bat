@echo off
:: Launches Windows Terminal in the current directory using the "PowerShell Preview" profile
:: and runs the python script.
start wt -p "PowerShell Preview" -d "%~dp0." pwsh -NoExit -Command "python pandoc_dashboard.py"
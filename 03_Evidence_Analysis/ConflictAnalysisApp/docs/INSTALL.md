# Install & Troubleshooting

## Dependencies

- Python 3.9+ recommended
- pip
- (Optional) Virtual environment

Install packages:

```bash
pip install -r requirements.txt
```

If PySide6 wheels fail on Linux, ensure system Qt deps are present or try:

```bash
pip install --upgrade pip setuptools wheel
pip install PySide6
```

If PDF text is empty:
- The export may be **image-based**. Use OCR (e.g., `ocrmypdf`) to convert before parsing.

If VADER sentiment not found:
```bash
pip install vaderSentiment
```

## Running

```bash
python app.py
```

## File Types

- **PDF**: Heuristic line parsing; prefer HTML/XML for accuracy.
- **HTML**: iMessage/Facebook-like exports (div.message + div.meta + p).
- **XML**: SMS Backup & Restore format.

## Windows Notes

- If long paths break, run `git config --system core.longpaths true` (if using git).

## macOS Notes

- Gatekeeper may block app execution in strict environments; run from Terminal.

## Linux Notes

- Wayland desktops may need `QT_QPA_PLATFORM=xcb` env var if rendering issues occur.

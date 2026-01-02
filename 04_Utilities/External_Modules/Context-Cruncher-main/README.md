# Context Cruncher 🎙️

Transform casual voice recordings into clean, structured context data for AI applications.

[![Demo](https://img.shields.io/badge/Demo-Live-brightgreen)](demo.html)
[![HuggingFace](https://img.shields.io/badge/🤗-Space-yellow)](https://huggingface.co/spaces/danielrosehill/Context-Cruncher)

## What is Context Cruncher?

Context Cruncher extracts structured context data from voice recordings using Gemini AI's multimodal capabilities. It processes audio directly, cleaning up natural speech patterns and organizing information into useful context data that AI systems can use for personalization.

**Context data** refers to specific information about users that grounds AI inference for more personalized results. This tool achieves that by:

- Removing irrelevant information and tangents
- Eliminating duplicates and redundancy
- Reformatting from first person to third person
- Organizing information hierarchically
- Outputting both Markdown and JSON formats

## See it in Action

Check out the [demo page](demo.html) to see real results from processing example audio about movie preferences.

## Features

- **🎤 Flexible Audio Input**: Record directly in your browser or upload audio files (MP3, WAV, OPUS)
- **🤖 AI-Powered Extraction**: Uses Gemini 2.0 Flash for intelligent audio understanding and context extraction
- **📝 Dual Output Formats**: Get both human-readable Markdown and machine-readable JSON
- **👤 Customizable Identification**: Choose how you're referred to in the context data (by name or as "the user")
- **📋 Easy Export**: Download files or copy directly to clipboard

## Quick Start

### Prerequisites

- Python 3.12+
- A [Gemini API key](https://ai.google.dev/)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/danielrosehill/Context-Cruncher.git
cd Context-Cruncher
```

2. Create a virtual environment and install dependencies:
```bash
# Using uv (recommended)
uv venv
source .venv/bin/activate
uv pip install -r requirements.txt

# Or using standard venv
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
pip install -r requirements.txt
```

3. Create a `.env` file with your Gemini API key:
```bash
cp .env.example .env
# Edit .env and add your API key:
# GEMINI_API="your_api_key_here"
```

4. Run the application:

**Option A: Using the launch script (easiest)**
```bash
./run.sh
```

**Option B: Manual launch**
```bash
source .venv/bin/activate
python app.py
```

The app will launch in your browser at `http://localhost:7860`

## Usage

1. **Configure**: Enter your Gemini API key (or load from `.env`)
2. **Choose Identification**: Select whether to be referred to by name or as "the user"
3. **Provide Audio**: Either:
   - Record directly in the browser using your microphone
   - Upload an audio file (MP3, WAV, or OPUS)
4. **Extract**: Click "Extract Context" to process your audio
5. **Download**: Get your structured context data as Markdown or JSON

## Example Transformation

**Raw Audio Input:**
> "Okay so... let's document my health problems and the meds I take for this AI project... ehm.. where do I start... well, I've had asthma since I was a kid. I take a daily inhaler called Relvar for that. I also take Vyvanse for ADHD which is a stimulant medication. Oh.. hey Jay! What's up, man! Yeah see you at the gym. Okay, where was I. Note to self, pick up the laundry later. Oh yeah.. I've been on Vyvanse for three years and think it's great. I get bloods every 3 months."

**Structured Output:**
```markdown
## Medical Conditions

- the user has had asthma since childhood
- the user has adult ADHD

## Medication List

- the user takes Relvar, daily, for asthma
- the user takes Vyvanse 70mg, daily, for ADHD
```

## Generating Demo Results

To regenerate the demo results with the example audio:

```bash
python generate_demo.py
```

This will process the `example-data/movie-prefs.opus` file and save results to `demo-results/`.

## Privacy Note

Your audio is processed using the Gemini API. Review [Google's privacy policies](https://policies.google.com/) before using this tool with sensitive information.

## Use Cases

- **AI Assistant Personalization**: Provide context to chatbots and AI assistants
- **Knowledge Management**: Convert verbal notes into structured information
- **Preference Mapping**: Document likes, dislikes, and preferences
- **Medical History**: Organize health information (note privacy considerations)
- **Project Context**: Capture project requirements and preferences

## Technical Details

- **Frontend**: Gradio web interface
- **AI Model**: Gemini 2.0 Flash (with multimodal audio understanding)
- **Audio Processing**: Direct audio file upload to Gemini API
- **Output Formats**: Markdown and JSON

## Repository Structure

```
Context-Cruncher/
├── app.py                  # Main Gradio application
├── gemini_processor.py     # Gemini API integration
├── generate_demo.py        # Demo generation script
├── run.sh                  # Launch script
├── requirements.txt        # Python dependencies
├── .env.example           # Environment variable template
├── demo.html              # Demo results page
├── example-data/          # Example audio files
└── demo-results/          # Generated demo outputs
```

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.

## License

MIT License - See LICENSE file for details

## Author

Daniel Rosehill
- Website: [danielrosehill.com](https://danielrosehill.com)
- GitHub: [@danielrosehill](https://github.com/danielrosehill)

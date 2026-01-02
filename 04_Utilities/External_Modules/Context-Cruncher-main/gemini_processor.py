"""
Gemini API integration for processing audio and extracting context data.
"""
import google.generativeai as genai
import json
from datetime import datetime
from typing import Dict, Tuple


def get_system_prompt(user_name: str = None) -> str:
    """
    Generate the system prompt for context extraction.

    Args:
        user_name: Optional name to use instead of "the user"

    Returns:
        System prompt string
    """
    user_reference = user_name if user_name else "the user"

    return f"""You are a context extraction assistant. Your task is to analyze audio recordings where users provide personal context information and extract it in a clean, structured format.

## Your Task

Extract context data from the user's audio recording. Context data refers to specific information about the user that can be used to ground AI inference for more personalized results.

## Transformation Guidelines

1. Remove irrelevant information (e.g., tangential conversations, notes to self)
2. Remove duplicates and redundancy
3. Reformat from first person to third person, referring to "{user_reference}"
4. Organize information hierarchically with clear sections
5. Present information in a clean, structured markdown format

## Example Transformation

INPUT (raw audio transcript):
"Okay so ... let's document my health problems and the meds I take for this AI project ... ehm.. where do i start ... well, I've had asthma since I was a kid. I take a daily inhaler called Relvar for that. I also take Vyvanse for ADHD which is a stimulant medication. Oh .. hey Jay! What's up, man! Yeah see you at the gym. Okay, where was I. Note to self, pick up the laundry later. Oh yeah .. I've been on Vyvanse for three years and think it's great. I get bloods every 3 months."

OUTPUT (cleaned context data):

## Medical Conditions

- {user_reference} has had asthma since childhood
- {user_reference} has adult ADHD

## Medication List

- {user_reference} takes Relvar, daily, for asthma
- {user_reference} takes Vyvanse 70mg, daily, for ADHD

## Important Notes

Follow a careful hierarchical structure that allows additional context to be easily integrated later. Use clear section headers and bullet points for organization.

Now process the provided audio recording and extract the context data following these guidelines."""


def get_naming_prompt() -> str:
    """Get the prompt for generating context data names."""
    return """Based on the context data you just extracted, provide a JSON object with:
1. human_readable_name: A clear, descriptive title for this context (e.g., "Medical History and Medications", "Movie Preferences")
2. snake_case_filename: A snake_case version suitable for a filename (e.g., "medical_history_medications", "movie_preferences")

Respond ONLY with a valid JSON object in this exact format:
{
  "human_readable_name": "Your Title Here",
  "snake_case_filename": "your_filename_here"
}"""


def process_audio_with_gemini(
    audio_file_path: str,
    api_key: str,
    user_name: str = None
) -> Tuple[str, str, str]:
    """
    Process audio file with Gemini API to extract context data.

    Args:
        audio_file_path: Path to the audio file
        api_key: Gemini API key
        user_name: Optional user name for personalization

    Returns:
        Tuple of (context_markdown, human_readable_name, snake_case_filename)

    Raises:
        Exception: If API call fails
    """
    genai.configure(api_key=api_key)

    # Use Gemini Pro 2.5 with audio understanding
    model = genai.GenerativeModel('gemini-2.0-flash-exp')

    # Upload the audio file
    audio_file = genai.upload_file(audio_file_path)

    # Generate context data
    system_prompt = get_system_prompt(user_name)
    response = model.generate_content([system_prompt, audio_file])
    context_markdown = response.text

    # Generate naming information
    naming_response = model.generate_content([
        context_markdown,
        get_naming_prompt()
    ])

    # Parse the JSON response
    try:
        # Extract JSON from response (handle potential markdown code blocks)
        naming_text = naming_response.text.strip()
        if naming_text.startswith('```'):
            # Remove markdown code block markers
            lines = naming_text.split('\n')
            naming_text = '\n'.join(lines[1:-1])

        naming_data = json.loads(naming_text)
        human_readable_name = naming_data['human_readable_name']
        snake_case_filename = naming_data['snake_case_filename']
    except (json.JSONDecodeError, KeyError) as e:
        # Fallback to generic naming if parsing fails
        human_readable_name = "Context Data"
        snake_case_filename = "context_data"

    return context_markdown, human_readable_name, snake_case_filename


def create_markdown_file(
    context_markdown: str,
    human_readable_name: str,
    snake_case_filename: str
) -> Tuple[str, str]:
    """
    Create a formatted markdown file content.

    Args:
        context_markdown: The extracted context data
        human_readable_name: Human readable title
        snake_case_filename: Filename

    Returns:
        Tuple of (filename, content)
    """
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    content = f"""## {human_readable_name}

{context_markdown}

---

Captured on: {timestamp}
"""

    filename = f"{snake_case_filename}.md"
    return filename, content


def create_json_file(
    context_markdown: str,
    human_readable_name: str,
    snake_case_filename: str
) -> Tuple[str, str]:
    """
    Create a JSON file content.

    Args:
        context_markdown: The extracted context data
        human_readable_name: Human readable title
        snake_case_filename: Filename

    Returns:
        Tuple of (filename, json_content)
    """
    timestamp = datetime.now().isoformat()

    data = {
        "human_readable_name": human_readable_name,
        "snake_case_filename": snake_case_filename,
        "context_data": context_markdown,
        "captured_on": timestamp
    }

    filename = f"{snake_case_filename}.json"
    json_content = json.dumps(data, indent=2)
    return filename, json_content

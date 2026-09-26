"""
Context Cruncher - Gradio Application
Extract structured context data from voice recordings using Gemini AI.
"""
import gradio as gr
import os
from pathlib import Path
import tempfile
from gemini_processor import (
    process_audio_with_gemini,
    create_markdown_file,
    create_json_file
)


def process_audio(
    audio_input,
    uploaded_file,
    api_key: str,
    user_identification: str,
    user_name: str = ""
) -> tuple:
    """
    Process audio from either recording or upload.

    Args:
        audio_input: Audio from microphone recording
        uploaded_file: Uploaded audio file
        api_key: Gemini API key
        user_identification: "name" or "user"
        user_name: User's name if using name identification

    Returns:
        Tuple of (markdown_content, markdown_file, json_file, status_message)
    """
    try:
        # Validate API key
        if not api_key or api_key.strip() == "":
            return (
                "",
                None,
                None,
                "Error: Please provide a Gemini API key"
            )

        # Determine which audio source to use
        audio_path = None
        if audio_input is not None:
            audio_path = audio_input
        elif uploaded_file is not None:
            audio_path = uploaded_file.name

        if audio_path is None:
            return (
                "",
                None,
                None,
                "Error: Please record audio or upload an audio file"
            )

        # Determine user reference
        user_ref = None
        if user_identification == "name":
            if not user_name or user_name.strip() == "":
                return (
                    "",
                    None,
                    None,
                    "Error: Please provide your name when using name identification"
                )
            user_ref = user_name.strip()

        # Process with Gemini
        status_msg = "Processing audio with Gemini API..."
        context_markdown, human_readable_name, snake_case_filename = process_audio_with_gemini(
            audio_path,
            api_key,
            user_ref
        )

        # Create output files
        md_filename, md_content = create_markdown_file(
            context_markdown,
            human_readable_name,
            snake_case_filename
        )

        json_filename, json_content = create_json_file(
            context_markdown,
            human_readable_name,
            snake_case_filename
        )

        # Write files to temp directory for download
        temp_dir = tempfile.mkdtemp()
        md_path = Path(temp_dir) / md_filename
        json_path = Path(temp_dir) / json_filename

        with open(md_path, 'w') as f:
            f.write(md_content)

        with open(json_path, 'w') as f:
            f.write(json_content)

        return (
            md_content,
            str(md_path),
            str(json_path),
            f"Success! Context extracted: {human_readable_name}"
        )

    except Exception as e:
        return (
            "",
            None,
            None,
            f"Error: {str(e)}"
        )


# Custom CSS for better styling
custom_css = """
.gradio-container {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
.main-header {
    text-align: center;
    margin-bottom: 1.5rem;
    padding-bottom: 1rem;
    border-bottom: 2px solid #e5e7eb;
}
.main-header h1 {
    font-size: 2rem;
    font-weight: 600;
    color: #1f2937;
    margin-bottom: 0.5rem;
}
.main-header p {
    color: #6b7280;
    font-size: 1rem;
}
.section-header {
    font-weight: 600;
    color: #374151;
    margin-bottom: 1rem;
}
"""

# Create Gradio interface
with gr.Blocks(css=custom_css, title="Context Cruncher") as demo:
    gr.Markdown(
        """
        # Context Cruncher

        Extract structured context data from voice recordings using AI
        """,
        elem_classes="main-header"
    )

    with gr.Tabs():
        with gr.Tab("Extract"):
            with gr.Row():
                with gr.Column(scale=1):
                    with gr.Accordion("Configuration", open=True):
                        api_key_input = gr.Textbox(
                            label="Gemini API Key",
                            placeholder="Enter your Gemini API key",
                            type="password",
                            info="Get your API key from https://ai.google.dev/"
                        )

                        user_identification = gr.Radio(
                            choices=["user", "name"],
                            value="user",
                            label="User Identification",
                            info="How should you be referred to in the context data?"
                        )

                        user_name_input = gr.Textbox(
                            label="Your Name",
                            placeholder="Enter your name",
                            visible=False,
                            info="Used when 'name' is selected above"
                        )

                    gr.Markdown("### Audio Input", elem_classes="section-header")

                    audio_recording = gr.Audio(
                        sources=["microphone"],
                        type="filepath",
                        label="Record Audio"
                    )

                    gr.Markdown("**OR**")

                    audio_upload = gr.File(
                        label="Upload Audio File",
                        file_types=["audio"],
                        type="filepath"
                    )

                    process_btn = gr.Button("Extract Context", variant="primary", size="lg")

                with gr.Column(scale=1):
                    gr.Markdown("### Results", elem_classes="section-header")

                    status_output = gr.Textbox(
                        label="Status",
                        interactive=False,
                        show_label=True
                    )

                    context_display = gr.Textbox(
                        label="Context Data (Markdown)",
                        lines=18,
                        interactive=False,
                        show_copy_button=True
                    )

                    with gr.Row():
                        markdown_download = gr.File(label="Download Markdown")
                        json_download = gr.File(label="Download JSON")

        with gr.Tab("About"):
            gr.Markdown(
                """
                ## What is Context Cruncher?

                Context Cruncher transforms casual voice recordings into clean, structured context data
                that AI systems can use for personalization.

                **Context data** refers to specific information about users that grounds AI inference
                for more personalized results.

                ## How It Works

                1. **Configure** - Enter your Gemini API key and choose how you want to be identified
                2. **Input Audio** - Either record directly in your browser or upload an audio file (MP3, WAV, OPUS)
                3. **Extract** - Click the button and let AI clean up your recording into structured context data
                4. **Download** - Get your context data as Markdown or JSON, or copy directly from the text area

                ## What Gets Extracted

                This tool processes your audio by:

                - Removing irrelevant information and tangents
                - Eliminating duplicates and redundancy
                - Reformatting from first person to third person
                - Organizing information hierarchically
                - Outputting both Markdown and JSON formats

                ## Example Transformation

                **Raw Audio:**
                > "Okay so... let's document my health problems... I've had asthma since I was a kid.
                > I take a daily inhaler called Relvar for that. Oh hey Jay! What's up!
                > Okay, where was I... I also take Vyvanse for ADHD."

                **Structured Output:**
                ```markdown
                ## Medical Conditions

                - the user has had asthma since childhood
                - the user has adult ADHD

                ## Medication List

                - the user takes Relvar, daily, for asthma
                - the user takes Vyvanse for ADHD
                ```

                ## Privacy Notice

                Your audio is processed using the Gemini API. Review Google's privacy policies
                before using this tool with sensitive information.

                ## Technical Details

                - **AI Model**: Gemini 2.0 Flash (multimodal audio understanding)
                - **Processing**: Direct audio file upload to Gemini API
                - **Output Formats**: Markdown and JSON

                ## Use Cases

                - AI assistant personalization
                - Knowledge management
                - Preference mapping
                - Medical history documentation (note privacy considerations)
                - Project context capture
                """
            )

    # Show/hide name input based on identification method
    def toggle_name_input(identification_choice):
        return gr.update(visible=identification_choice == "name")

    user_identification.change(
        fn=toggle_name_input,
        inputs=[user_identification],
        outputs=[user_name_input]
    )

    # Process button click
    process_btn.click(
        fn=process_audio,
        inputs=[
            audio_recording,
            audio_upload,
            api_key_input,
            user_identification,
            user_name_input
        ],
        outputs=[
            context_display,
            markdown_download,
            json_download,
            status_output
        ]
    )


if __name__ == "__main__":
    demo.launch()

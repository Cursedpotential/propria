This repository contains a utility that I have found very useful in various AI workflows - a utility that extracts "context data" from user-supplied voice notes.

Here's the context:

Voice is a great way to capture information quickly and I believe that it lends itself perfectly to a workflow that aims to capture user-specific context data proactively rather than passively (e.g. through building up memory over time).

I use this approach when I have a lot of data to provide for a specific project. As I'm open-sourcing this repo, I've chosen a movie recommendations context recording (example-data) which aims to provide some information about the type of video content I enjoy. 

The task in this repo: create a Hugging Face space. This will be for public use and I will clone it for my personal implementation. 

LLM method: BYOK with Gemini Pro 2.5. Gemini has audio understanding. 

API documentation is here: https://ai.google.dev/gemini-api/docs/audio

## Record Audio

A panel for the user to either: 

1) Record audio from the browser (with controls for record, pause, stop, abort).
2) Upload an audio file (accepting opus, mp3, wav) 

A button to extract content. 

## Context Extraction Logic

"Context data," in this context, refers to specific information about the user that can be used to ground AI inference to produce more personalised results.

Context data can be achieved from a typical STT transcription through:

- Omitting irrelevant information 
- Removing duplicates 
- Reformatting from the first person to the third, referring to "the user."

Here is a short example to model the desired type of transformation, expecting that a raw STT transcript would contain approximately this level of defect:

"Okay so ... let's document my health problems and the meds I take for this AI project ... ehm.. where do i start ... well, I've had asthma since I was a kid. I take a daily inhaler called Relvar for that. I also take Vyvanse for ADHD which is a stimulant medication. Oh .. hey Jay! What's up, man! Yeah see you at the gym. Okay, where was I. Note to self, pick up the laundry later. Oh yeah .. I've been on Vyvanse for three years and think it's great. I get bloods every 3 months."

This text would be transformed to (approximately):

{START EXAMPLE}

## Medical Conditions

- User has had asthma since childhood  
- User has adult ADHD 

## Medication List 

- User takes Relvar, daily, for asthma  
- User takes Vyvanse 70mg, daily, for ADHD

{END EXAMPLE}

The above model is written with the idea that if the user were to later provide more detailed context, an agent could easily slot it into the right places. Hence, follow a careful hierarchical structure when formatting the context notes.

Most of my context parsing utilities to date have followed a two part approach: STT then LLM cleanup (which is why I'm creating an updated interface as Gemini multimodal means that these steps can be combined!).

In this updated interface the audio binary should be sent to Gemini alongside the system prompt (ensuring to include the example).

The context data which is extracted can be provided to the user as a markdown file that populates in a right hand pane called Extracted Context

## User Identification

I would like to provide one user editable parameter: how they would like to be identified in the context data.

That is to say: should the user be referred to by name ("Daniel has asthma") or referred to as "the user."

If by name, I would like the user to be able to provide their name. 

This information should be carried from the frontend into the system prompt constructed in the API call to Gemini. 

## Context Download Options

The intended use is that after the context data is parsed, the user can download it. I would like to offer the user the ability to download it as a markdown doc or as JSON - we may as well just generate both and provide buttons. We should also provide a clipboard on the markdown body text as this will provide an easy way to grab the output that doesn't require any download.

I would like the context data to have a unique name. The logical place to implement this is again with Gemini API.

So we should ask it for a JSON output that provides:

- Human readable name 
- Snake case filename 
- Context data (as markdown) 

to this we can add a timestamp.

## Template for markdown file

For the markdown version the filename can be the snakecase file name.md

The actual contents can follow this template

{Start Example}
## Readable Context Title

{Context Data}

---

Captured on: {timestamp}

{End Example}

## JSON Template

For the JSON version we should construct an object with all these parameters. 

For both, after generation, they should be presented to the user with download buttons.
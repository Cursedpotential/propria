# Provenance and upstream material

The model library combines two source groups preserved from the original `think.zip` package:

- **36 underlying model bodies** originating from TJ Boudreaux's `thinking-skills` / `cc-thinking-skills` 1.0.0 material, covered by the bundled MIT license.
- **48 owner models** carried forward from the owner's existing thinking-model library.

The original package also contained three meta skills (router, model selection, and model combination). In 2.0, selection and combination behavior is consolidated into the single public `thinking-model-router`; the former meta-skill text is retained only as legacy source material under that router's `references/` directory.

For the 84 underlying models, the canonical prior model body is retained under each model's **Full Process** section. Version 2.0 adds a portable progressive-depth wrapper (Micro / Reflex, Focused, Full) and routing metadata around that body. Therefore the model files are no longer byte-for-byte copies of their sources.

The current package layout—not historical cache/install paths—is authoritative for discovery and loading.

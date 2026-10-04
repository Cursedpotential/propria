"""Reference wrapper: ColQwen2.5 behind the HTTP contract the image index's ``image_colqwen`` \
slot expects.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

NOT DEPLOYED and NOT RUN: written from the colpali-engine and Modal documentation and not \
exercised on a GPU here.
It exists so the slot can be switched on without writing the server first. To use it (owner step):

    modal secret create colqwen-token COLQWEN_API_TOKEN=<a long random string>
    modal deploy deploy/colqwen_modal.py          # prints the https URL of the ``api`` function
    # then in the Coolify environment of the superindex app:
    #   INTAKE_IMAGES_COLQWEN_URL=<that URL>   COLQWEN_API_TOKEN=<the same string>
    #   INTAKE_IMAGES_SLOTS=image_maxsim,image_single,image_colqwen   (and recreate the \
collection: new named vector)

Contract (``image_embedders.ColQwenEmbedder``): ``POST /embed_image {"image": <base64>}`` and
``POST /embed_query {"text": <str>}``, each ``{"embeddings": [[128 floats], ...]}``; bearer \
token required.
Cost model (assumed throughput, unmeasured): Modal L4 is $0.000222 per second; at 3 to 5 images \
per second that is about
$0.04 to $0.07 per 1,000 images, billed only while a container runs (scale to zero after \
``scaledown_window``).
Images are sent over HTTPS to the owner's own Modal account; nothing else sees them.
"""

import base64
import io
import os

import modal

MODEL_ID = "vidore/colqwen2.5-v0.2"

app = modal.App("colqwen-embed")
image = (
    modal.Image.debian_slim(python_version="3.12")
    .pip_install("colpali-engine>=0.3.8", "torch", "transformers", "pillow", "fastapi[standard]")
    .env({"HF_HOME": "/cache"})
)
cache = modal.Volume.from_name("colqwen-hf-cache", create_if_missing=True)


@app.cls(gpu="L4", image=image, volumes={"/cache": cache}, \
secrets=[modal.Secret.from_name("colqwen-token")],
         scaledown_window=120, max_containers=1)
class ColQwen:
    """One L4 container holding the model; ``api`` is the ASGI app."""

    @modal.enter()
    def load(self) -> None:
        import torch
        from colpali_engine.models import ColQwen2_5, ColQwen2_5_Processor

        self.torch = torch
        self.model = ColQwen2_5.from_pretrained(MODEL_ID, torch_dtype=torch.bfloat16, \
device_map="cuda").eval()
        self.processor = ColQwen2_5_Processor.from_pretrained(MODEL_ID)

    @modal.asgi_app()
    def api(self):
        from fastapi import Depends, FastAPI, Header, HTTPException
        from PIL import Image

        web = FastAPI()

        def guard(authorization: str = Header(default="")) -> None:
            if authorization != f"Bearer {os.environ['COLQWEN_API_TOKEN']}":
                raise HTTPException(status_code=401)

        @web.post("/embed_image", dependencies=[Depends(guard)])
        def embed_image(body: dict) -> dict:
            picture = Image.open(io.BytesIO(base64.b64decode(body["image"]))).convert("RGB")
            batch = self.processor.process_images([picture]).to(self.model.device)
            with self.torch.no_grad():
                out = self.model(**batch)
            return {"embeddings": out[0].float().cpu().tolist()}

        @web.post("/embed_query", dependencies=[Depends(guard)])
        def embed_query(body: dict) -> dict:
            batch = self.processor.process_queries([body["text"]]).to(self.model.device)
            with self.torch.no_grad():
                out = self.model(**batch)
            return {"embeddings": out[0].float().cpu().tolist()}

        return web
